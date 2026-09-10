# ECS EC2 with Terraform

| Field | Value |
| --- | --- |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |
| **Platform** | `ecs` |

## Overview

Deploys SigNoz onto an ECS cluster you already run (EC2 launch type), using
Terraform. Each component becomes its own ECS service and finds the others
through AWS Cloud Map.

Components:
- ClickHouse Keeper (telemetry keeper)
- ClickHouse (telemetry store)
- PostgreSQL (metadata store)
- SigNoz (UI + API server on port 8080)
- OTel Collector (ingester)
- Schema migrator (one-shot task)

The example beside this file is [`byo/`](byo/): the cluster is yours, and every
object it is made of is stated on the casting.

> [!IMPORTANT]
> Two limits to know before you deploy this.
>
> **No persistence.** Every volume is task-scoped, so ClickHouse, the keeper and
> PostgreSQL lose their data when their task is replaced. Pinning a task to the
> instance holding its disk is what makes a host path safe, and nothing here
> does that yet: without the pin, a task that restarts elsewhere finds a clean
> disk and initialises fresh, silently.
>
> **One node per component.** `spec.telemetrystore.spec.cluster.shards` and
> `spec.telemetrykeeper.spec.cluster.replicas` are not honoured yet; the pour is
> a single ClickHouse node and a single keeper regardless.

## Prerequisites

- An ECS cluster with registered EC2 container instances
- A VPC with private subnets, and a security group that permits the intra-cluster
  traffic in the table at the end of this file
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.4

## Configuration

Nothing about the cluster is discovered. Each object is named on the casting,
arrives in Terraform as a variable defaulted to what was stated, and is used
as-is.

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:111122223333:cluster/observability
    foundry.signoz.io/ecs-vpc-id: vpc-0a1b2c3d4e5f67890
    foundry.signoz.io/ecs-subnet-ids: subnet-0a1b2c3d4e5f67890,subnet-0f9e8d7c6b5a43210
    foundry.signoz.io/ecs-security-group-ids: sg-0a1b2c3d4e5f67890
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
```

Leave one out and the forge refuses, rather than rendering an empty string into
Terraform. The two IAM roles are the exception: they are this stack's own
identity, so an absent one is created here rather than looked up.

### Component config

Component config reaches a task through AWS AppConfig. Terraform lifts each
YAML file out of the pour into a hosted configuration version, and an
`aws-appconfig-agent` sidecar writes it into the task at start and on every
poll. The YAML in `pours/` stays the source of truth and stays
`spec.patches`-targetable.

ClickHouse and the keeper reload a rewritten config in place. The ingester reads
its config once at start, so its task definition carries a digest of that config
and a change replaces the task.

### Metadata store

`spec.metastore.kind: sqlite` runs no PostgreSQL service; SigNoz holds the file
itself, and its service is clamped to one task. The default, `postgres`, runs a
service.

## Deploy

```bash
foundryctl cast -f casting.yaml
```

> [!NOTE]
> `cast` runs `terraform init` then `terraform apply -auto-approve`. To review
> the plan first, use the steps below.

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
  backend.tf.json
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

One root module, one file per component. There are no child modules: a module
with a single generated caller is indirection without reuse, and module paths
rewrite state addresses.

State lives beside the configuration that declares it, which is Terraform's own
default. `backend.tf.json` states it anyway, so moving to a remote backend is a
patch rather than an edit to a generated file:

```yaml
spec:
  patches:
  - target: "deployment/backend.tf.json"
    type: jsonpatch
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

Reach the UI by pointing an ALB at the SigNoz service on port 8080, and send
telemetry through an NLB on 4317/4318.

## Annotations

| Annotation | Required | Names |
| --- | --- | --- |
| `foundry.signoz.io/ecs-region` | yes | the region holding the cluster |
| `foundry.signoz.io/ecs-cluster-arn` | yes | the cluster |
| `foundry.signoz.io/ecs-subnet-ids` | yes | the subnets tasks are placed in |
| `foundry.signoz.io/ecs-security-group-ids` | yes | the security groups tasks join |
| `foundry.signoz.io/ecs-vpc-id` | yes | the VPC the Cloud Map namespace is created in |
| `foundry.signoz.io/ecs-task-role-arn` | no | a task role to use instead of creating one |
| `foundry.signoz.io/ecs-task-execution-role-arn` | no | an execution role to use instead of creating one |

The ID annotations take a comma-separated list.

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
```

Run `foundryctl forge` first to read the generated file and find the path you
want.

File names and Terraform resource labels are a public surface: renaming one
breaks every stored patch, and for resource labels, live state addresses too.

## Platform details

### Variables

Every identifier is a variable defaulted to what the casting stated, so a
one-off change needs no edit to a generated file. Change `casting.yaml` and the
default moves with it, or pass `-var` for a single apply.

| Variable | Default |
| --- | --- |
| `aws_region` | from the region annotation |
| `cluster_arn` | from the cluster annotation |
| `subnet_ids` | from the subnet annotation |
| `security_group_ids` | from the security group annotation |
| `vpc_id` | from the VPC annotation |
| `task_role_name` | `<name>-installation-iam-task` |
| `execution_role_name` | `<name>-installation-iam-exec` |

The role names come from `metadata.name`, not the cluster, so several
installations can share one cluster.

### Resources

| Resource | Count | Purpose |
| --- | --- | --- |
| `aws_service_discovery_private_dns_namespace` | 1 | `<name>.local` |
| `aws_service_discovery_service` | one per component | DNS record |
| `aws_ecs_service` | one per component | long-running |
| `aws_ecs_task_definition` | components + 1 | the extra is the migrator |
| `aws_appconfig_*` | one set per config file | component config |
| `aws_iam_role` | 2 | task and execution, unless stated |
| `aws_ecs_task_execution` (data) | 1 | runs the migrator once, on Fargate |

Every service carries a deployment circuit breaker with rollback, so a revision
that never becomes healthy puts the previous one back instead of sitting there.

### Container logs

Every container writes through the `json-file` driver with rotation
(`max-size: 10m`, `max-file: 3`) and copies four ECS labels into each log line,
so a collection agent reading the instance can attribute a line to the task it
came from. The migrator is the exception: it runs on Fargate, which has no
`json-file` driver, so its logs are not readable from the instance.

### Service discovery

Components resolve each other inside `<name>.local`:

| Component | DNS name | Port |
| --- | --- | --- |
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper-0` | 9181 client, 9234 raft |
| ClickHouse | `telemetrystore-clickhouse-0-0` | 9000 |
| PostgreSQL | `metastore-postgres-0` | 5432 |
| SigNoz | `signoz` | 8080 API, 4320 OpAMP |
| Ingester | `ingester` | 4317 gRPC, 4318 HTTP |

### IAM

The **execution role** pulls images and writes logs
(`AmazonECSTaskExecutionRolePolicy`).

The **task role** reads component config from AppConfig:
`appconfig:StartConfigurationSession`, scoped to the configurations of the
application this stack creates, and `appconfig:GetLatestConfiguration`, which
acts on a session token and cannot be scoped.

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
