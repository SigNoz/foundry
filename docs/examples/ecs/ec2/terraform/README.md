# ECS EC2 Installation

| Field | Value |
| --- | --- |
| **Kind** | `Installation` |
| **Platform** | `ecs` |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |

## Overview

This casting deploys SigNoz onto an ECS cluster on the EC2 launch type. `foundryctl` generates the Terraform and applies it.

It creates:

- One ECS service and task definition per node of ClickHouse Keeper or ZooKeeper, ClickHouse, PostgreSQL, SigNoz, the ingester, and MCP when enabled
- A Cloud Map private DNS namespace named `<metadata.name>-installation.local`, with one record per node
- An AppConfig application, with one configuration profile per config file
- The task and execution IAM roles, unless the casting states them
- The schema migrator, run once as a Fargate task

How to describe each component is in [Moldings](../../../../concepts/moldings.md).

> [!IMPORTANT]
> On a cluster you bring, mount a durable volume at `/var/lib/foundry` on the instances that run stateful services, and pin each stateful service to the instance holding its data yourself. A [bound Infrastructure](#binding-to-provisioned-infrastructure) does both.

## Prerequisites

- [foundryctl](../../../../getting-started.md)
- An ECS cluster with registered EC2 container instances
- A VPC with private subnets, and a security group that permits traffic between the components on the ports listed under [Validate](#validate)
- AWS credentials in the environment Terraform runs in
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer

## Configuration

On a cluster you bring, name the region and every object the cluster is made of through an annotation:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-private-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
```

What each annotation names, and when it is required, is in [Annotations](#annotations). The two IAM roles are created unless you state your own.

`spec.metastore.kind` defaults to `postgres`, which runs as its own ECS service. Under `sqlite`, SigNoz holds the database file itself, in place of a PostgreSQL service.

Each stateful node keeps its data under `/var/lib/foundry/<name>/<component>/<kind>/<node>`, for example `/var/lib/foundry/signoz/telemetrystore/clickhouse/0-0`.

### Topology

Shard and replica counts become nodes, each its own ECS service with its own Cloud Map record. This casting runs four ClickHouse, three ClickHouse Keeper, two SigNoz and two PostgreSQL nodes:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-private-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
  telemetrystore:
    spec:
      cluster:
        shards: 2
        replicas: 1  # copies beside each shard, so four nodes here
  telemetrykeeper:
    spec:
      cluster:
        replicas: 3  # keep it odd, for quorum
  signoz:
    spec:
      cluster:
        replicas: 2  # under sqlite each node holds its own database
  metastore:
    spec:
      cluster:
        replicas: 2
```

### Binding to provisioned infrastructure

An Infrastructure document can provision the cluster instead. Bind the installation to it by setting `spec.infrastructure.name` to the Infrastructure's `metadata.name`.

> [!IMPORTANT]
> Bind and state none of the four cluster annotations, or state all four and leave `spec.infrastructure` out; the forge refuses a mix.

The [Infrastructure example](../../../infrastructure/aws/ecs/terraform/) provisions infrastructure named `foundry`; [Deploy](#deploy) covers cast order.

Each cluster object is looked up at plan by the name or tags the Infrastructure gave it, and an empty lookup fails the plan:

| Object | Looked up by |
| --- | --- |
| ECS cluster | Name `<infrastructure>-cls` |
| VPC | Tag `foundry.signoz.io/name: <infrastructure>` |
| Private subnets | Tags `foundry.signoz.io/name: <infrastructure>` and `foundry.signoz.io/subnet-type: private`, in that VPC |
| Task security group | Name `<infrastructure>-sg-task` and tag `foundry.signoz.io/name: <infrastructure>`, in that VPC |

Every service is placed by the `foundry.signoz.io/storage` attribute the Infrastructure's container instances advertise, and stays pending until an instance of its class registers:

| `foundry.signoz.io/storage` | Components |
| --- | --- |
| `persistent` | ClickHouse, ClickHouse Keeper or ZooKeeper, PostgreSQL, and SigNoz under `sqlite` |
| `ephemeral` | Ingester, MCP, and SigNoz under `postgres` |

Each stateful node claims a persistent data volume through the volume's `foundry.signoz.io/identities` tag, a comma-separated list of the nodes it holds, such as `telemetrystore-clickhouse-0-0`.

| Event | What happens |
| --- | --- |
| First cast | Each stateful node claims a volume, and its service is pinned to the instance that volume is attached to |
| Instance replaced | The next plan finds the instance each claimed volume is attached to now, and the replacement takes over the nodes on its volume |
| Node added | It takes an unclaimed volume first, and shares a claimed one once every volume is claimed |
| Volume detached | Its nodes stay pending until the volume is attached again |
| Node removed | Its name stays in the volume's tag, so the node returns to its own data if added again; releasing the volume is under [Changing the installation](#changing-the-installation) |

> [!NOTE]
> A bound installation still states `foundry.signoz.io/ecs-region`, set to the region the Infrastructure runs in, because the lookups run there.

The bound installation's `casting.yaml`, whose `spec.infrastructure.name` must equal the Infrastructure document's `metadata.name`:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
  infrastructure:
    name: foundry
```

### MCP server

Set `spec.mcp.spec.enabled` to `true` to deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server), which adds `mcp.tf.json`, an ECS service and a Cloud Map record at `mcp.<name>-installation.local` on port `8000`. Connecting an AI client is in [MCP server](../../../../concepts/mcp-server.md).

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-private-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
  mcp:
    spec:
      enabled: true
```

## Deploy

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
foundryctl cast -f casting.yaml
```

`gauge` checks that Terraform is installed and usable. `forge` writes the Terraform under `pours/deployment/` and the resolved casting to `casting.yaml.lock`. `cast` runs `terraform init`, `terraform plan -out=tfplan` and `terraform apply tfplan` in `pours/deployment/`.

> [!IMPORTANT]
> A bound installation's plan fails until its Infrastructure is cast. In a shared casting file Foundry casts the Infrastructure document first, whatever order they are written in; with separate files, cast the Infrastructure file first.

With both documents in one casting file, the forge writes one root per document:

```text
pours/
  infrastructure/   # the Infrastructure document's Terraform
  deployment/       # this installation's Terraform
```

A document that fails stops the run, and what already cast stays cast.

## Generated output

```text
pours/deployment/                   # one root module, one file per component, no child modules
  versions.tf.json                  # required Terraform and provider versions
  providers.tf.json                 # the aws provider, in var.aws_region
  backend.tf.json                   # local state, in this directory
  main.tf.json                      # the Cloud Map namespace, the AppConfig application and the roles; when bound, the lookups and the claims
  variables.tf.json                 # the region, the cluster objects and the roles the root takes
  terraform.tfvars.json             # the values the casting resolved from its annotations
  outputs.tf.json                   # the cluster, namespace, subnet, security group and service identifiers
  telemetrykeeper.tf.json           # ClickHouse Keeper: a service, task definition and Cloud Map record per node, and their configuration profiles
  telemetrystore.tf.json            # ClickHouse: a service, task definition and Cloud Map record per node, and their configuration profiles
  telemetrystore_migrator.tf.json   # the schema migrator, run once on Fargate
  metastore.tf.json                 # PostgreSQL: a service, task definition and Cloud Map record per node
  signoz.tf.json                    # SigNoz: a service, task definition and Cloud Map record per node
  ingester.tf.json                  # the ingester's service, task definition, Cloud Map record and configuration profiles
  telemetrykeeper/
    clickhousekeeper/
      keeper-0.yaml                 # Keeper node 0's config, uploaded to AppConfig
  telemetrystore/
    clickhouse/
      config-0-0.yaml               # ClickHouse node 0-0's config, uploaded to AppConfig
      functions.yaml                # ClickHouse's user-defined functions, uploaded to AppConfig
  ingester/
    ingester.yaml                   # the collector config, uploaded to AppConfig
    opamp.yaml                      # the collector's OpAMP connection to SigNoz, uploaded to AppConfig
```

Terraform keeps its state in `pours/deployment/terraform.tfstate`. Keep that file, because `terraform destroy` needs it. If you forge into another location, copy the file across, or move the state to a remote backend with a patch on `backend.tf.json`:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-private-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
  patches:
    - target: "deployment/backend.tf.json"
      operations:
        - op: replace
          path: /terraform/backend
          value:
            s3:
              bucket: foundry-tfstate
              key: signoz/deployment.tfstate
              region: us-east-1
```

## Validate

```bash
# The services in the cluster the casting names: one per node
aws ecs list-services --cluster observability --region us-east-1

# The Cloud Map services: one per node
aws servicediscovery list-services --region us-east-1
```

Components resolve each other inside `<name>-installation.local`:

| Component | DNS name | Ports |
| --- | --- | --- |
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper-<i>` | 9181 client, 9234 raft |
| ZooKeeper | `telemetrykeeper-zookeeper-<i>` | 2181 client, 2888 raft, 3888 election, 9141 metrics |
| ClickHouse | `telemetrystore-clickhouse-<shard>-<replica>` | 9000 native, 8123 HTTP, 9009 interserver, 9363 metrics |
| PostgreSQL | `metastore-postgres-<i>` | 5432 |
| SigNoz | `signoz-<i>` | 8080 API, 4320 OpAMP |
| Ingester | `ingester` | 4317 gRPC, 4318 HTTP |
| MCP | `mcp` | 8000 |

## Changing the installation

Edit `casting.yaml`, then forge and cast again.

- **Configuration**: delivered through [AWS AppConfig](https://docs.aws.amazon.com/appconfig/) and reloaded in place; the ingester restarts.
- **Stateful node removed**, when bound: the plan refuses a volume left holding only removed nodes until you release it with the note below.
- **CPU, memory and other platform-level settings**: [patch](../../../../concepts/patches.md) the generated file for the component; forge first and read that file to find the path.

> [!NOTE]
> Release the volume by removing its claim from the state at the address the plan names, then cast; the tag stays on the volume until the Infrastructure removes it.
>
> ```bash
> cd pours/deployment
> terraform state rm 'aws_ec2_tag.claims["<volume id>"]'
> ```

This casting gives the first SigNoz node 1024 CPU units:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-private-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    flavor: terraform
    mode: ec2
    platform: ecs
  patches:
    - target: "deployment/signoz.tf.json"
      operations:
        - op: replace
          path: /locals/containers_signoz_0/0/cpu
          value: 1024
```

## Teardown

> [!IMPORTANT]
> On a bound installation, a plain `terraform destroy` refuses while the claims are in its state, so remove them from the state first. The claims stay on the volumes, and the next cast pins each node back to its own data.
>
> ```bash
> cd pours/deployment
> terraform state rm aws_ec2_tag.claims
> terraform destroy
> ```

On a cluster you bring:

```bash
cd pours/deployment && terraform destroy
```

This removes the services and everything else the root created.

Destroying the Infrastructure removes the volumes and their claims together. Destroy the Infrastructure after the installation.

## Annotations

A required annotation left out reaches Terraform as an empty value, and the plan refuses it.

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region holding the cluster. Required. |
| `foundry.signoz.io/ecs-cluster-arn` | ARN of the ECS cluster. Required on a cluster you bring; left out when bound. |
| `foundry.signoz.io/ecs-private-subnet-ids` | Comma-separated IDs of private subnets, each with a NAT route or the ECR, ECS, S3, AppConfig and logs VPC endpoints, since tasks on the EC2 launch type take no public IP; the plan refuses a subnet that assigns public IPs on launch. Required on a cluster you bring; left out when bound. |
| `foundry.signoz.io/ecs-security-group-ids` | Comma-separated security group IDs. Required on a cluster you bring; left out when bound. |
| `foundry.signoz.io/ecs-vpc-id` | VPC ID for the Cloud Map namespace. Required on a cluster you bring; left out when bound. |
| `foundry.signoz.io/ecs-task-role-arn` | IAM role the tasks assume; it must allow `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration`. Created when absent. |
| `foundry.signoz.io/ecs-task-execution-role-arn` | IAM role the ECS agent assumes to pull images and start tasks. Created when absent. |

The same annotations, with the tfvar each one sets, are in the [casting file reference](../../../../reference/casting-file.md#ecs-annotations).
