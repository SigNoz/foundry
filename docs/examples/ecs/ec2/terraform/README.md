# ECS EC2 with Terraform

| Field | Value |
| --- | --- |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |
| **Platform** | `ecs` |

## Overview

Deploys SigNoz on AWS ECS (EC2 launch type) using Terraform. Each component runs as its own ECS service and finds the others through AWS Cloud Map.

Components:
- ClickHouse Keeper (telemetry keeper)
- ClickHouse (telemetry store)
- PostgreSQL (metadata store)
- SigNoz (UI + API server on port 8080)
- OTel Collector (ingester)
- Schema migrator (one-shot task)

Two examples sit beside this file, one per way of naming the cluster:

| | [`provisioned/`](provisioned/) | [`byo/`](byo/) |
| --- | --- | --- |
| The cluster | from the Infrastructure casting | already yours |
| Objects are | found by tag | stated on the casting |
| Placement | pinned to the instance holding the data | none, ECS chooses |

## Prerequisites

- An ECS cluster with registered EC2 container instances
- A VPC with private subnets
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.4

For `provisioned/`, instances must advertise `foundry.signoz.io/name` and `foundry.signoz.io/storage`, and persistent ones must mount a durable volume at `/var/lib/foundry`. The Infrastructure casting does both.

## Configuration

`spec.infrastructure.name` names the substrate. Everything about the cluster is then found by the tags both castings derive from that one name, and arrives in Terraform as a variable defaulted to what was derived. The region is the only thing left to state, because nothing else carries it.

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: foundry
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
  infrastructure:
    name: foundry
```

### Bringing your own cluster

Any identifier can be stated instead of found. A stated one is used as-is and no data source is emitted for it; the rest are still found by tag. You can adopt a substrate and pin one piece of it.

Leave `spec.infrastructure.name` out and there is no substrate to find anything by, so the cluster, VPC, subnets and security group must all be stated. The pour then has no data source, no claim and no placement constraint. Stateful components still run, but nothing pins them to the instance holding their volume, so their data does not survive being rescheduled.

```yaml
metadata:
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
```

### Multi-node clusters

ClickHouse and the keeper can run as a multi-node cluster:

```yaml
spec:
  telemetrykeeper:
    spec:
      cluster:
        replicas: 3        # odd number, for raft quorum
  telemetrystore:
    spec:
      cluster:
        shards: 2
        replicas: 1        # in addition to the primary, so 2 nodes per shard
```

Each node gets its own ECS service, Cloud Map service and task definition. ClickHouse nodes are `shards x (replicas+1)`, named `telemetrystore-clickhouse-<shard>-<replica>`; keeper nodes are `replicas`, named `telemetrykeeper-<kind>-<index>`. Every node fetches its own config file.

### Placement and claims

Stateful components place onto instances advertising `storage == persistent`, stateless ones onto `ephemeral`.

An identity claims a volume, not an instance. Terraform reads the `foundry.signoz.io/identities` tag off the persistent volumes: claims that exist stay put, new identities take unclaimed volumes and then wrap. The claim is written back as a tag, and each stateful service is pinned to whichever instance currently holds its volume. Because the record lives on the volume, re-forging never moves data, an instance replacement re-attaches, and an operator can move an identity by editing the tag.

With fewer instances than identities they share machines, and the ECS scheduler decides what fits; the rest stay PENDING rather than starting on the wrong disk.

## Deploy

```bash
foundryctl cast -f casting.yaml
```

> [!NOTE]
> `cast` runs `terraform init` then `terraform apply -auto-approve`. To review the plan first, use the steps below.

```bash
foundryctl forge -f casting.yaml
cd pours/deployment
terraform init
terraform apply
```

## Generated output

```text
pours/deployment/
  versions.tf.json
  providers.tf.json
  main.tf.json
  variables.tf.json
  outputs.tf.json
  terraform.tfvars.json
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

One root module, one file per component. There are no child modules: a module with a single generated caller is indirection without reuse, and module paths rewrite state addresses.

## After deployment

```bash
# Services
aws ecs list-services --cluster foundry-cls --region us-east-1

# Service discovery
aws servicediscovery list-services --region us-east-1
```

Reach the UI by pointing an ALB at the SigNoz service on port 8080, and send telemetry through an NLB on 4317/4318.

## Annotations

Only the region is required. The rest each replace one lookup.

| Annotation | Replaces | With |
| --- | --- | --- |
| `foundry.signoz.io/ecs-region` | | required |
| `foundry.signoz.io/ecs-cluster-arn` | `cluster_name` | `cluster_arn` |
| `foundry.signoz.io/ecs-subnet-ids` | `subnet_tags` | `subnet_ids` |
| `foundry.signoz.io/ecs-security-group-ids` | `security_group_name` | `security_group_ids` |
| `foundry.signoz.io/ecs-vpc-id` | `vpc_tags` | `vpc_id` |
| `foundry.signoz.io/ecs-task-role-arn` | `task_role_name` | `task_role_arn` |
| `foundry.signoz.io/ecs-task-execution-role-arn` | `execution_role_name` | `execution_role_arn` |

## Customization

Defaults for CPU and memory are patched on the generated file for the component:

```yaml
spec:
  patches:
  - target: "deployment/signoz.tf.json"
    type: jsonpatch
    operations:
      - op: replace
        path: /locals/containers_signoz/0/cpu
        value: 1024
      - op: replace
        path: /locals/containers_signoz/0/memory
        value: 1024
```

Run `foundryctl forge` first to read the generated file and find the path you want.

File names and Terraform resource labels are a public surface: renaming one breaks every stored patch, and for resource labels, live state addresses too.

## Platform details

### Variables

Everything resolved about the substrate is a variable defaulted to what was resolved, so a one-off change needs no edit to a generated file. Change `casting.yaml` and the default moves with it, or pass `-var` for a single apply.

| Variable | Default |
| --- | --- |
| `aws_region` | from the region annotation |
| `cluster_name` | `<substrate>-cls` |
| `subnet_tags` | `name=<substrate>`, `subnet-type=private` |
| `security_group_name` | `<substrate>-sg-task` |
| `vpc_tags` | `name=<substrate>` |
| `task_role_name` | `<name>-installation-task` |
| `execution_role_name` | `<name>-installation-exec` |
| `node_tags` | `name=<substrate>`, `storage=persistent` |
| `claim_tag` | `foundry.signoz.io/identities` |

The role names come from the casting, not the substrate, so several installations can share one cluster.

### Resources

| Resource | Count | Purpose |
| --- | --- | --- |
| `aws_service_discovery_private_dns_namespace` | 1 | `<name>.local` |
| `aws_service_discovery_service` | one per service | DNS record |
| `aws_ecs_service` | one per component | long-running |
| `aws_ecs_task_definition` | components + 1 | the extra is the migrator |
| `aws_appconfig_*` | one set per config file | component config, read at task start |
| `aws_iam_role` | 2 | task and execution, unless stated |
| `aws_ec2_tag` | 1 | writes the claim back |
| `aws_ecs_task_execution` (data) | 1 | runs the migrator once, on EC2 |

### Service discovery

Components resolve each other inside `<name>.local`:

| Component | DNS name | Port |
| --- | --- | --- |
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper-<i>` | 9181 client, 9234 raft |
| ClickHouse | `telemetrystore-clickhouse-<shard>-<replica>` | 9000 |
| PostgreSQL | `metastore-postgres-0` | 5432 |
| SigNoz | `signoz` | 8080 API, 4320 OpAMP |
| Ingester | `ingester` | 4317 gRPC, 4318 HTTP |

### IAM

The **execution role** pulls images and writes logs (`AmazonECSTaskExecutionRolePolicy`).

The **task role** reads component config from AppConfig: `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration`.

### Security groups

Services use `awsvpc`. The group must allow:

| From | To | Port |
| --- | --- | --- |
| Ingester, SigNoz | ClickHouse | 9000 |
| SigNoz | PostgreSQL | 5432 |
| SigNoz | Ingester | 4320 |
| ClickHouse | Keeper | 9181 |
| ALB | SigNoz | 8080 |
| NLB | Ingester | 4317, 4318 |
