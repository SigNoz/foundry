# ECS EC2 with Terraform

| | |
|---|---|
| Deployment Platform | `ecs` |
| Deployment Mode | `ec2` |
| Deployment Flavor | `terraform` |
| Use Case | AWS ECS on EC2 with Terraform-managed infrastructure |

## Overview

Deploys SigNoz on AWS ECS (EC2 launch type) using Terraform. Each component runs as a separate ECS service with AWS Cloud Map for service discovery.

Components:
- ClickHouse Keeper (telemetry keeper)
- ClickHouse (telemetry store)
- PostgreSQL (metadata store)
- SigNoz (UI + API server)
- OTel Collector (ingester)
- Schema migrator (Fargate one-shot task)

## Prerequisites

- An existing ECS cluster with an EC2 capacity provider
- A VPC with private subnets
- An S3 bucket for storing component configs
- IAM roles for ECS task and task execution
- Terraform >= 1.0

## Quick Start

### With Foundry

```yaml
# casting.yaml
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

#### Metadata Annotations

Annotations populate `terraform.tfvars.json` so Foundry can generate a ready-to-apply Terraform configuration:

| Annotation | Maps to tfvar | Description |
|---|---|---|
| `foundry.signoz.io/ecs/region` | `region` | AWS region |
| `foundry.signoz.io/ecs/cluster-id` | `ecs_cluster_id` | ECS cluster ARN or ID |
| `foundry.signoz.io/ecs/subnet-ids` | `subnet_ids` | Comma-separated subnet IDs |
| `foundry.signoz.io/ecs/security-group-ids` | `security_group_ids` | Comma-separated security group IDs |
| `foundry.signoz.io/ecs/vpc-id` | `vpc_id` | VPC ID for Cloud Map namespace |
| `foundry.signoz.io/ecs/config-bucket` | `config_bucket` | S3 bucket for component configs |
| `foundry.signoz.io/ecs/task-role-arn` | `task_role_arn` | IAM role ARN for ECS tasks |
| `foundry.signoz.io/ecs/task-execution-role-arn` | `task_execution_role_arn` | IAM role ARN for task execution |
| `foundry.signoz.io/ecs/capacity-provider` | `capacity_provider` | ECS capacity provider name |

The `log_group` is auto-generated as `/ecs/{metadata.name}`.

#### Deployment Spec

| Field | Value | Description |
|---|---|---|
| `spec.deployment.platform` | `ecs` | Target platform (AWS ECS) |
| `spec.deployment.mode` | `ec2` | Launch type (EC2 instances, not Fargate) |
| `spec.deployment.flavor` | `terraform` | Infrastructure-as-code tool |

#### Run

```bash
# Generate Terraform files only
foundryctl forge

# Or generate and apply in one step (runs terraform init + apply)
foundryctl cast
```

Foundry generates the Terraform files into `pours/deployment/`. You can also run `forge` first and apply manually:

```bash
cd pours/deployment
terraform init
terraform apply
```

## Architecture

```text
pours/deployment/
  main.tf.json              # Root module: provider, module call
  variables.tf.json         # Root variables (passed through to module)
  terraform.tfvars.json     # User-provided values
  module/
    main.tf.json            # CloudWatch log group, Cloud Map namespace
    variables.tf.json       # Module input variables
    outputs.tf.json         # Service ARNs, namespace info
    telemetrykeeper.tf.json # ClickHouse Keeper: task def, service, SD
    telemetrystore.tf.json  # ClickHouse: task def, service, SD, S3 configs
    telemetrystore_migrator.tf.json  # Schema migrator (Fargate task)
    metastore.tf.json       # PostgreSQL: task def, service, SD
    signoz.tf.json          # SigNoz: task def, service, SD
    ingester.tf.json        # OTel Collector: task def, service, SD
```

## Providers

| Provider | Version | Purpose |
|---|---|---|
| `hashicorp/aws` | `>= 5.0` | ECS, Cloud Map, CloudWatch, S3 |

## Resources

The module creates the following AWS resources:

| Resource | Count | Description |
|---|---|---|
| `aws_service_discovery_private_dns_namespace` | 1 | Cloud Map namespace (`{name}.local`) |
| `aws_cloudwatch_log_group` | 1 | Shared log group for all ECS services |
| `aws_ecs_task_definition` | 6 | One per component (including migrator) |
| `aws_ecs_service` | 5 | One per long-running component |
| `aws_service_discovery_service` | 5 | One per long-running component |
| `aws_s3_object` | N | Config files for ClickHouse, Keeper, and Ingester |
| `aws_ecs_task_execution` (data) | 1 | Runs the migrator as a Fargate task |

## Variables

### Required

| Variable | Type | Description |
|---|---|---|
| `region` | `string` | AWS region |
| `ecs_cluster_id` | `string` | ID of the existing ECS cluster |
| `subnet_ids` | `list(string)` | Subnet IDs for ECS service networking (awsvpc) |
| `security_group_ids` | `list(string)` | Security group IDs for ECS service networking |
| `vpc_id` | `string` | VPC ID for the Cloud Map private DNS namespace |
| `config_bucket` | `string` | S3 bucket for storing component config files |
| `task_role_arn` | `string` | IAM role ARN for ECS tasks |
| `task_execution_role_arn` | `string` | IAM role ARN for ECS task execution (pull images, write logs) |
| `log_group` | `string` | CloudWatch log group name |
| `capacity_provider` | `string` | Name of the ECS capacity provider |

### Optional

| Variable | Type | Default | Description |
|---|---|---|---|
| `log_retention_days` | `number` | `30` | CloudWatch log retention in days |
| `telemetrykeeper_cpu` | `number` | `256` | CPU units for ClickHouse Keeper |
| `telemetrykeeper_memory` | `number` | `512` | Memory (MiB) for ClickHouse Keeper |
| `telemetrystore_cpu` | `number` | `1024` | CPU units for ClickHouse |
| `telemetrystore_memory` | `number` | `512` | Memory (MiB) for ClickHouse |
| `migrator_cpu` | `number` | `256` | CPU units for schema migrator (Fargate) |
| `migrator_memory` | `number` | `512` | Memory (MiB) for schema migrator (Fargate) |
| `metastore_cpu` | `number` | `256` | CPU units for PostgreSQL |
| `metastore_memory` | `number` | `256` | Memory (MiB) for PostgreSQL |
| `signoz_cpu` | `number` | `512` | CPU units for SigNoz |
| `signoz_memory` | `number` | `512` | Memory (MiB) for SigNoz |
| `ingester_cpu` | `number` | `512` | CPU units for OTel Collector |
| `ingester_memory` | `number` | `512` | Memory (MiB) for OTel Collector |

## Outputs

| Output | Description |
|---|---|
| `namespace_id` | Cloud Map private DNS namespace ID |
| `namespace_name` | Cloud Map private DNS namespace name |
| `signoz_service_arn` | SigNoz ECS service ARN (target for ALB on port 8080) |
| `signoz_service_name` | SigNoz ECS service name |
| `ingester_service_arn` | Ingester ECS service ARN (target for NLB on port 4317/4318) |
| `ingester_service_name` | Ingester ECS service name |
| `telemetrystore_service_name` | ClickHouse ECS service name |
| `telemetrykeeper_service_name` | ClickHouse Keeper ECS service name |
| `metastore_service_name` | PostgreSQL ECS service name |

## Service Discovery

Components communicate via Cloud Map DNS within the `{name}.local` namespace:

| Component | DNS name | Port |
|---|---|---|
| ClickHouse Keeper | `telemetrykeeper-clickhousekeeper.{name}.local` | 9181 (client), 9234 (raft) |
| ClickHouse | `telemetrystore-clickhouse.{name}.local` | 9000 (native), 8123 (HTTP) |
| PostgreSQL | `metastore-postgresql.{name}.local` | 5432 |
| SigNoz | `signoz.{name}.local` | 8080 (API), 4320 (OpAMP) |
| Ingester | `ingester.{name}.local` | 4317 (gRPC), 4318 (HTTP) |


## IAM Requirements

The **task execution role** (`task_execution_role_arn`) needs:
- `ecr:GetAuthorizationToken`, `ecr:BatchGetImage`, `ecr:GetDownloadUrlForLayer` (pull images)
- `logs:CreateLogStream`, `logs:PutLogEvents` (CloudWatch logs)

The **task role** (`task_role_arn`) needs:
- `s3:GetObject` on the config bucket (config-fetcher sidecar reads configs from S3)

## Security Groups

ECS services use `awsvpc` networking. Security groups must allow:

| From | To | Port | Purpose |
|---|---|---|---|
| Ingester | ClickHouse | 9000 | Telemetry writes |
| SigNoz | ClickHouse | 9000 | Query reads |
| SigNoz | PostgreSQL | 5432 | Metadata |
| SigNoz | Ingester | 4320 | OpAMP management |
| ClickHouse | ClickHouse Keeper | 9181 | Coordination |
| External | SigNoz | 8080 | UI/API access (via ALB) |
| External | Ingester | 4317, 4318 | Telemetry ingestion (via NLB) |
