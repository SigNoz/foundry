# ECS EC2 with Terraform

| Field | Value |
| --- | --- |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |
| **Platform** | `ecs` |

## Overview

Deploys SigNoz on AWS ECS (EC2 launch type) using Terraform. Each component runs as a separate ECS service with AWS Cloud Map for service discovery.

Components:
- ClickHouse Keeper (telemetry keeper)
- ClickHouse (telemetry store)
- PostgreSQL (metadata store)
- SigNoz (UI + API server on port 8080)
- OTel Collector (ingester)
- Schema migrator (Fargate one-shot task)

## Prerequisites

- An ECS cluster tagged the way foundry names things, either from the Infrastructure casting or your own (see [Infrastructure contract](#infrastructure-contract-you-provide-the-cluster))
- Registered EC2 container instances advertising the substrate's attributes
- A VPC with private subnets
- IAM roles for ECS task and task execution
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## Configuration

`spec.infrastructure.name` names the substrate this installation runs on. That
one field is the whole binding: the cluster, its subnets, its security group,
its VPC and its two IAM roles are all found by the names and tags both castings
derive from it, and each arrives in Terraform as a variable defaulted to what
was derived. The region is the only thing left to state; neither the casting
nor a tag can carry it.

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
  infrastructure:
    name: signoz
```

### Bringing your own cluster

Each identifier can be stated instead of discovered, one at a time. What you
state is used verbatim and no lookup is emitted for it; what you leave out is
still discovered. You can adopt a substrate and pin one piece of it.

```yaml
metadata:
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster
    foundry.signoz.io/ecs-subnet-ids: subnet-abc123,subnet-def456
    foundry.signoz.io/ecs-security-group-ids: sg-abc123
    foundry.signoz.io/ecs-vpc-id: vpc-abc123
    foundry.signoz.io/ecs-task-role-arn: arn:aws:iam::123456789012:role/ecs-task-role
    foundry.signoz.io/ecs-task-execution-role-arn: arn:aws:iam::123456789012:role/ecs-execution-role
```

`spec.infrastructure.name` is still required: the container instances and their
volumes are always found by tag, whoever provisioned them.

## Multi-node clusters

ClickHouse (telemetry store) and the keeper can run as a multi-node cluster. Set the cluster sizes in the spec:

```yaml
spec:
  telemetrykeeper:
    spec:
      cluster:
        replicas: 3        # keeper nodes; use an odd number for raft quorum
  telemetrystore:
    spec:
      cluster:
        shards: 2          # number of shards
        replicas: 1        # replicas in addition to the primary, so 2 nodes per shard
```

This generates one ECS service, one Cloud Map service, and one task definition **per node**: ClickHouse nodes are `shards x (replicas+1)` (named `telemetrystore-clickhouse-<shard>-<replica>`) and keeper nodes are `replicas` (named `telemetrykeeper-<kind>-<index>`). Each ClickHouse node fetches its own `config-<shard>-<replica>.yaml`; each keeper node fetches its own `keeper-<index>.yaml` (ClickHouse Keeper) or is clustered via `ZOO_SERVER_ID` / `ZOO_SERVERS` env (ZooKeeper).

Each stateful task places onto the persistent storage class and bind-mounts its data under `/var/lib/foundry/<name>/...`; stateless tasks place onto the ephemeral pool. Tasks use `launch_type = EC2`, so ECS places them onto your registered container instances:

| Component | Placed on instances advertising |
| --- | --- |
| ClickHouse, Keeper, Postgres | `foundry.signoz.io/name == <substrate>` and `foundry.signoz.io/storage == persistent` |
| SigNoz with `sqlite` | `foundry.signoz.io/name == <substrate>` and `foundry.signoz.io/storage == persistent` |
| Ingester, SigNoz with `postgres` | `foundry.signoz.io/name == <substrate>` and `foundry.signoz.io/storage == ephemeral` |

Node identity is claimed, not computed. At plan time, Terraform reads the `foundry.signoz.io/identities` tag off the persistent instances: identities already claimed stay exactly where they are, new identities take unclaimed instances first and then wrap round-robin onto the fleet. At apply, the claim is written back as the tag and each stateful service is pinned with `ec2InstanceId == '<claimed instance>'` (plus the storage attribute as a bootstrap check). The claim record lives on the instances themselves: neither foundry nor its lock ever holds a binding, re-forging never moves data, and an operator can move an identity by editing the tag with plain AWS knowledge.

Placement is best effort by design: with at least two persistent instances, replicas of one shard land on distinct machines (identities assign round-robin in shard-major order); with fewer instances than identities, they share machines and the ECS scheduler's own limits (task ENIs, memory reservations) decide what actually fits, leaving the rest PENDING. A replaced instance loses its tag, so its identities re-claim automatically on the next apply and start on a fresh disk (replicated components resync; unreplicated start empty).

### Infrastructure contract (you provide the cluster)

The Installation kind **deploys onto** an ECS cluster; it does not provision compute or storage. Whether you use the Infrastructure kind or bring your own cluster, it must satisfy the same convention:

- **Registered EC2 container instances**, enough for the topology: one persistent instance per stateful node. `shards: 2 / replicas: 1` + `3` keepers + Postgres is `4 + 3 + 1 = 8` persistent instances; SigNoz and Ingester run on `ephemeral` capacity.
- **Instances advertising `foundry.signoz.io/name` and `foundry.signoz.io/storage`** (`persistent` or `ephemeral`). Set them via the agent's `ECS_INSTANCE_ATTRIBUTES={"foundry.signoz.io/name":"signoz","foundry.signoz.io/storage":"persistent"}` or `aws ecs put-attributes`. The name is what keeps one installation off another's nodes in a shared cluster. A task with no matching instance stays PENDING (fail-loud, never silent data loss).
- **Instances and volumes tagged the same way**. The claim controller finds them by tag, not from a list it is handed.
- **A durable volume (e.g. EBS) mounted at `/var/lib/foundry`** on each `persistent` instance, so bind-mounted data survives restarts. Keep the volume's lifecycle independent of the instance (a standalone EBS volume re-attached to a replacement) and the node's data survives termination too.

Data survives task restarts on the same instance. A task rescheduled onto a different persistent instance starts empty and re-replicates from its peers (or, for an unreplicated component, starts fresh). Counts, sizing, and autoscaling are yours to manage; foundry only wires the deployment.

Leaving the cluster blocks unset deploys a single node of each component (the default).

## Deploy

Run the full pipeline (generate Terraform files and apply):

```bash
foundryctl cast -f casting.yaml
```

> [!NOTE]
> `foundryctl cast` runs `terraform init` followed by `terraform apply -auto-approve`. If you prefer to review the plan before applying, use the step-by-step approach below.

Step-by-step alternative:

```bash
# 1. Generate Terraform files
foundryctl forge -f casting.yaml

# 2. Initialize and apply Terraform
cd pours/deployment
terraform init
terraform apply
```

## Generated output

```text
pours/deployment/
  main.tf.json
  variables.tf.json
  terraform.tfvars.json
  module/
    main.tf.json
    variables.tf.json
    outputs.tf.json
    telemetrykeeper.tf.json
    telemetrystore.tf.json
    telemetrystore_migrator.tf.json
    metastore.tf.json
    signoz.tf.json
    ingester.tf.json
    telemetrykeeper/
      clickhousekeeper/
        keeper-0.yaml
    telemetrystore/
      clickhouse/
        config.yaml
        functions.yaml
    ingester/
      ingester.yaml
      opamp.yaml
```

## After deployment

Verify the ECS services are running:

```bash
aws ecs list-services --cluster signoz-cls --region us-east-1
aws ecs describe-services \
  --cluster signoz-cls \
  --services signoz-signoz signoz-ingester signoz-telemetrystore-clickhouse \
  --region us-east-1
```

Check that Cloud Map service discovery is healthy:

```bash
aws servicediscovery list-services --region us-east-1
```

Access the SigNoz UI by setting up an ALB pointing to the SigNoz service on port 8080.

## Customization

The module ships with sensible defaults for CPU and memory. To override them, use `spec.patches` on the generated module files:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    # ... (same annotations as above)
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
  patches:
  - target: "deployment/module/signoz.tf.json"
    type: jsonpatch
    operations:
      - op: replace
        path: /locals/containers/0/cpu
        value: 1024
      - op: replace
        path: /locals/containers/0/memory
        value: 1024
      - op: replace
        path: /locals/containers/0/memoryReservation
        value: 1024
  - target: "deployment/module/telemetrystore.tf.json"
    type: jsonpatch
    operations:
      - op: replace
        path: /locals/containers/2/cpu
        value: 2048
      - op: replace
        path: /locals/containers/2/memory
        value: 4096
      - op: replace
        path: /locals/containers/2/memoryReservation
        value: 4096
```

Run `foundryctl forge` to see the generated files and identify the JSON paths you want to patch.

## Annotations

Only the region is required. The rest override one discovered identifier each,
and are for a cluster foundry did not provision.

| Annotation | Replaces the lookup variable | With |
| --- | --- | --- |
| `foundry.signoz.io/ecs-region` | | required; nothing else carries it |
| `foundry.signoz.io/ecs-cluster-arn` | `cluster_name` | `cluster_arn` |
| `foundry.signoz.io/ecs-subnet-ids` | `subnet_tags` | `subnet_ids` |
| `foundry.signoz.io/ecs-security-group-ids` | `security_group_name` | `security_group_ids` |
| `foundry.signoz.io/ecs-vpc-id` | `vpc_tags` | `vpc_id` |
| `foundry.signoz.io/ecs-task-role-arn` | `task_role_name` | `task_role_arn` |
| `foundry.signoz.io/ecs-task-execution-role-arn` | `execution_role_name` | `execution_role_arn` |

## Platform details

### Providers

| Provider | Version | Purpose |
| --- | --- | --- |
| `hashicorp/aws` | `>= 5.0` | ECS, Cloud Map, S3 |

### Resources

The module creates the following AWS resources:

| Resource | Count | Description |
| --- | --- | --- |
| `aws_service_discovery_private_dns_namespace` | 1 | Cloud Map namespace (`{name}.local`) |
| `aws_ecs_task_definition` | 6 | One per component (including migrator) |
| `aws_ecs_service` | 5 | One per long-running component |
| `aws_service_discovery_service` | 5 | One per long-running component |
| `aws_s3_object` | N | Config files for ClickHouse, Keeper, and Ingester |
| `aws_ecs_task_execution` (data) | 1 | Runs the migrator as a Fargate task |

### Variables

Everything the casting resolved about the substrate is a variable whose default
is what it resolved. A one-off change needs no edit to a generated file.
Change `casting.yaml` and the default moves with it; pass `-var` to override a
single apply.

| Variable | Type | Default |
| --- | --- | --- |
| `aws_region` | `string` | from the region annotation |
| `cluster_name` | `string` | `<substrate>-cls` |
| `subnet_tags` | `map(string)` | `name=<substrate>`, `visibility=private` |
| `security_group_name` | `string` | `<substrate>-sg-task` |
| `vpc_tags` | `map(string)` | `name=<substrate>` |
| `task_role_name` | `string` | `<substrate>-iam-task` |
| `execution_role_name` | `string` | `<substrate>-iam-exec` |
| `node_tags` | `map(string)` | `name=<substrate>`, `storage=persistent` |
| `claim_tag` | `string` | `foundry.signoz.io/identities` |

Stating an identifier on the casting swaps its lookup variable for the value
itself: `cluster_name` becomes `cluster_arn`, `subnet_tags` becomes
`subnet_ids`, and no data source is emitted for it.

### Outputs

| Output | Description |
| --- | --- |
| `namespace_id` | Cloud Map private DNS namespace ID |
| `namespace_name` | Cloud Map private DNS namespace name |
| `signoz_service_arn` | SigNoz ECS service ARN (target for ALB on port 8080) |
| `signoz_service_name` | SigNoz ECS service name |
| `ingester_service_arn` | Ingester ECS service ARN (target for NLB on port 4317/4318) |
| `ingester_service_name` | Ingester ECS service name |
| `telemetrystore_service_name` | ClickHouse ECS service name |
| `telemetrykeeper_service_name` | ClickHouse Keeper ECS service name |
| `metastore_service_name` | PostgreSQL ECS service name |

### Service discovery

Components communicate via Cloud Map DNS within the `{name}.local` namespace:

| Component | DNS name | Port |
| --- | --- | --- |
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper.{name}.local` | 9181 (client), 9234 (raft) |
| ClickHouse | `telemetrystore-clickhouse.{name}.local` | 9000 (native), 8123 (HTTP) |
| PostgreSQL | `metastore-postgresql.{name}.local` | 5432 |
| SigNoz | `signoz.{name}.local` | 8080 (API), 4320 (OpAMP) |
| Ingester | `ingester.{name}.local` | 4317 (gRPC), 4318 (HTTP) |

### IAM requirements

The **task execution role** (`execution_role_name`) needs:
- `ecr:GetAuthorizationToken`, `ecr:BatchGetImage`, `ecr:GetDownloadUrlForLayer` (pull images)
- `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch logs)

The **task role** (`task_role_name`) needs:
- `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration` (the config sidecar reads each component's config from AWS AppConfig)

### Security groups

ECS services use `awsvpc` networking. Security groups must allow:

| From | To | Port | Purpose |
| --- | --- | --- | --- |
| Ingester | ClickHouse | 9000 | Telemetry writes |
| SigNoz | ClickHouse | 9000 | Query reads |
| SigNoz | PostgreSQL | 5432 | Metadata |
| SigNoz | Ingester | 4320 | OpAMP management |
| ClickHouse | ClickHouse Keeper | 9181 | Coordination |
| External | SigNoz | 8080 | UI/API access (via ALB) |
| External | Ingester | 4317, 4318 | Telemetry ingestion (via NLB) |
