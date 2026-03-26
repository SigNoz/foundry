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

- An existing ECS cluster with an EC2 capacity provider
- A VPC with private subnets
- An S3 bucket for storing component configs
- IAM roles for ECS task and task execution
- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs/region: us-east-1
    foundry.signoz.io/ecs/cluster-id: arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster
    foundry.signoz.io/ecs/subnet-ids: subnet-abc123,subnet-def456
    foundry.signoz.io/ecs/security-group-ids: sg-abc123
    foundry.signoz.io/ecs/vpc-id: vpc-abc123
    foundry.signoz.io/ecs/config-bucket: my-signoz-configs
    foundry.signoz.io/ecs/task-role-arn: arn:aws:iam::123456789012:role/ecs-task-role
    foundry.signoz.io/ecs/task-execution-role-arn: arn:aws:iam::123456789012:role/ecs-execution-role
    foundry.signoz.io/ecs/capacity-provider: my-capacity-provider
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
```

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
aws ecs list-services --cluster my-cluster --region us-east-1
aws ecs describe-services \
  --cluster my-cluster \
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

Annotations populate `terraform.tfvars.json` so Foundry can generate a ready-to-apply Terraform configuration.

| Annotation | Maps to tfvar | Description |
| --- | --- | --- |
| `foundry.signoz.io/ecs/region` | `region` | AWS region |
| `foundry.signoz.io/ecs/cluster-id` | `ecs_cluster_id` | ECS cluster ARN or ID |
| `foundry.signoz.io/ecs/subnet-ids` | `subnet_ids` | Comma-separated subnet IDs |
| `foundry.signoz.io/ecs/security-group-ids` | `security_group_ids` | Comma-separated security group IDs |
| `foundry.signoz.io/ecs/vpc-id` | `vpc_id` | VPC ID for Cloud Map namespace |
| `foundry.signoz.io/ecs/config-bucket` | `config_bucket` | S3 bucket for component configs |
| `foundry.signoz.io/ecs/task-role-arn` | `task_role_arn` | IAM role ARN for ECS tasks |
| `foundry.signoz.io/ecs/task-execution-role-arn` | `task_execution_role_arn` | IAM role ARN for task execution |
| `foundry.signoz.io/ecs/capacity-provider` | `capacity_provider` | ECS capacity provider name |

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

| Variable | Type | Description |
| --- | --- | --- |
| `region` | `string` | AWS region |
| `ecs_cluster_id` | `string` | ID of the existing ECS cluster |
| `subnet_ids` | `list(string)` | Subnet IDs for ECS service networking (awsvpc) |
| `security_group_ids` | `list(string)` | Security group IDs for ECS service networking |
| `vpc_id` | `string` | VPC ID for the Cloud Map private DNS namespace |
| `config_bucket` | `string` | S3 bucket for storing component config files |
| `task_role_arn` | `string` | IAM role ARN for ECS tasks |
| `task_execution_role_arn` | `string` | IAM role ARN for ECS task execution (pull images, write logs) |
| `capacity_provider` | `string` | Name of the ECS capacity provider |

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

The **task execution role** (`task_execution_role_arn`) needs:
- `ecr:GetAuthorizationToken`, `ecr:BatchGetImage`, `ecr:GetDownloadUrlForLayer` (pull images)
- `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch logs)

The **task role** (`task_role_arn`) needs:
- `s3:GetObject` on the config bucket (config-fetcher sidecar reads configs from S3)

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
