# ECS EC2 with Terraform

| Field | Value |
| --- | --- |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |
| **Platform** | `ecs` |

## Overview

Deploys SigNoz onto an ECS cluster you already run, on the EC2 launch type, using Terraform. Each component becomes its own ECS service and finds the others through AWS Cloud Map.

Components:
- ClickHouse Keeper (telemetry keeper)
- ClickHouse (telemetry store)
- PostgreSQL (metadata store)
- SigNoz (UI and API server on port 8080)
- OTel Collector (ingester)
- Schema migrator, run once on Fargate

The example beside this file is [`byo/`](byo/): the cluster is yours, and every object it is made of is stated on the casting.

> [!IMPORTANT]
> Mount a durable volume at `/var/lib/foundry` on the instances that run stateful services, otherwise data lives on the instance's root disk. This deployment sets no placement constraints; pin stateful services to their instances yourself.
>
> One node per component: cluster shard and replica counts are not honoured yet.

## Prerequisites

- An ECS cluster with registered EC2 container instances
- A VPC with private subnets, and a security group that permits traffic between the components on the ports listed under [After deployment](#after-deployment)
- AWS credentials in the environment Terraform runs in
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer

## Configuration

Nothing about the cluster is discovered. Name every object on the casting through an annotation:

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
    platform: ecs
    mode: ec2
    flavor: terraform
```

The forge refuses if a required annotation is missing. The ID annotations take a comma-separated list. The subnets must be private: the root reads each stated subnet at plan and refuses one that assigns public IPs on launch. For the full annotation table, see the [casting file reference](../../../reference/casting-file.md).

The two IAM roles are optional: state `foundry.signoz.io/ecs-task-role-arn` or `foundry.signoz.io/ecs-task-execution-role-arn` to use your own, otherwise they are created here. A stated task role must allow `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration`.

Component configuration is delivered through AWS AppConfig and reloaded in place; the ingester restarts on a change.

`spec.metastore.kind` defaults to `postgres`, which runs as its own ECS service. Under `sqlite` no PostgreSQL service is created and SigNoz holds the database file itself.

Stateful data lives on the instance running the task, under `/var/lib/foundry/<name>/<component>/<kind>/<node>`, for example `/var/lib/foundry/signoz/telemetrystore/clickhouse/0-0`.

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
foundryctl forge -f casting.yaml
cd pours/deployment
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

## Generated output

```text
pours/deployment/
  versions.tf.json
  providers.tf.json
  backend.tf.json
  main.tf.json
  variables.tf.json
  terraform.tfvars.json
  outputs.tf.json
  telemetrykeeper.tf.json
  telemetrystore.tf.json
  telemetrystore_migrator.tf.json
  metastore.tf.json
  signoz.tf.json
  ingester.tf.json
  telemetrykeeper/clickhousekeeper/keeper-0.yaml
  telemetrystore/clickhouse/config-0-0.yaml
  telemetrystore/clickhouse/functions.yaml
  ingester/ingester.yaml
  ingester/opamp.yaml
```

One root module, one file per component, and no child modules. It creates:

| Resource | Count |
| --- | --- |
| `aws_service_discovery_private_dns_namespace` | 1, named `<name>-installation.local` |
| `aws_service_discovery_service` | one per component |
| `aws_ecs_service` | one per component |
| `aws_ecs_task_definition` | one per component, plus the migrator |
| `aws_appconfig_*` | one set per config file |
| `aws_iam_role` | 2, unless stated on the casting |

State is written to `terraform.tfstate` beside the configuration. To move it to a remote backend, patch `backend.tf.json`:

```yaml
spec:
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

## After deployment

```bash
# Services
aws ecs list-services --cluster observability --region us-east-1

# Service discovery
aws servicediscovery list-services --region us-east-1
```

Components resolve each other inside `<name>-installation.local`:

| Component | DNS name | Ports |
| --- | --- | --- |
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper-0` | 9181 client, 9234 raft |
| ZooKeeper | `telemetrykeeper-zookeeper-0` | 2181 client, 2888 raft, 3888 election, 9141 metrics |
| ClickHouse | `telemetrystore-clickhouse-0-0` | 9000 native, 8123 HTTP, 9009 interserver, 9363 metrics |
| PostgreSQL | `metastore-postgres-0` | 5432 |
| SigNoz | `signoz-0` | 8080 API, 4320 OpAMP |
| Ingester | `ingester` | 4317 gRPC, 4318 HTTP |
| MCP | `mcp` | 8000 |

## MCP server (optional)

Foundry can deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) alongside the stack so AI clients can query your telemetry. Set `spec.mcp.spec.enabled` to `true` and the pour gains an `mcp.tf.json` file, an ECS service, and a Cloud Map record at `mcp.<name>-installation.local` on port `8000`.

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

Change CPU, memory or any other platform-level setting by patching the generated file for the component:

```yaml
spec:
  patches:
    - target: "deployment/signoz.tf.json"
      operations:
        - op: replace
          path: /locals/containers_signoz_0/0/cpu
          value: 1024
```

Run `foundryctl forge` first, then read the generated file to find the path you want. See [patches](../../../concepts/patches.md).