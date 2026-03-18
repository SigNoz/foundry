# ECS EC2 Casting

| | |
|---|---|
| Deployment Mode | `ecs` / `ec2` |
| Use Case | AWS ECS on EC2 instances |

## Overview

Deploys SigNoz as separate ECS services on EC2 instances. Each component runs as its own ECS service with its own task definition, connected via [ECS Service Connect](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-connect.html) for DNS-based service discovery.

### Services

Foundry generates 6 per-component task definitions:

| Service | Containers | Service Connect Ports |
|---|---|---|
| `telemetrykeeper` | config-fetcher, clickhouse-keeper | `keeper-client:9181`, `keeper-raft:9234` |
| `telemetrystore` | init-clickhouse, config-fetcher, clickhouse | `clickhouse-native:9000`, `clickhouse-http:8123` |
| `telemetrystore-migrator` | telemetrystore-migrator | : |
| `metastore` | metastore-postgres | `postgres:5432` |
| `signoz` | signoz | `signoz-http:8080`, `signoz-opamp:4320` |
| `ingester` | config-fetcher, otel-collector | `otel-grpc:4317`, `otel-http:4318` |

## Prerequisites

### Tools

- [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) configured with credentials
- `foundryctl` binary

### AWS Infrastructure Setup

The following AWS resources must be created before deploying. Replace placeholder values (`my-ecs-cluster`, `123456789012`, etc.) with your own.

#### 1. ECS Cluster

Create an ECS cluster with Container Insights enabled:

```bash
aws ecs create-cluster \
  --cluster-name my-ecs-cluster \
  --settings name=containerInsights,value=enabled \
  --region us-east-1
```

If using a capacity provider (recommended), create and attach one:

```bash
aws ecs create-capacity-provider \
  --name my-capacity-provider \
  --auto-scaling-group-provider "autoScalingGroupArn=arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:xxx:autoScalingGroupName/my-asg,managedScaling={status=ENABLED,targetCapacity=100},managedTerminationProtection=ENABLED" \
  --region us-east-1

aws ecs put-cluster-capacity-providers \
  --cluster my-ecs-cluster \
  --capacity-providers my-capacity-provider \
  --default-capacity-provider-strategy capacityProvider=my-capacity-provider,weight=1,base=0 \
  --region us-east-1
```

You also need EC2 instances registered to the cluster (via an Auto Scaling Group with the ECS-optimized AMI and the `ECS_CLUSTER=my-ecs-cluster` user data).

#### 2. VPC, Subnets, and Security Groups

Use an existing VPC or create one. Private subnets are recommended.

Create a security group that allows inter-service traffic within the cluster:

```bash
aws ec2 create-security-group \
  --group-name signoz-ecs-sg \
  --description "Security group for SigNoz ECS services" \
  --vpc-id vpc-xxxxxxxx \
  --region us-east-1

# Allow all traffic within the security group (inter-service communication)
aws ec2 authorize-security-group-ingress \
  --group-id sg-abc123 \
  --protocol -1 \
  --source-group sg-abc123 \
  --region us-east-1
```

Note the subnet IDs and security group ID : you'll need them when creating ECS services.

#### 3. IAM Roles

**Task Execution Role** : allows the ECS agent to pull container images and write CloudWatch logs:

```bash
aws iam create-role \
  --role-name ecsTaskExecutionRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }'

aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy
```

**Task Role** : allows running containers to access S3 (config bucket) and enables ECS Exec:

```bash
aws iam create-role \
  --role-name ecsTaskRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }'

aws iam put-role-policy \
  --role-name ecsTaskRole \
  --policy-name signoz-ecs-task-policy \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": ["s3:GetObject", "s3:ListBucket"],
        "Resource": [
          "arn:aws:s3:::my-signoz-config-bucket",
          "arn:aws:s3:::my-signoz-config-bucket/*"
        ]
      },
      {
        "Effect": "Allow",
        "Action": [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel"
        ],
        "Resource": "*"
      },
      {
        "Effect": "Allow",
        "Action": [
          "logs:CreateLogStream",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams",
          "logs:PutLogEvents"
        ],
        "Resource": "*"
      }
    ]
  }'
```

#### 4. S3 Config Bucket

Create a bucket to store ClickHouse, ClickHouse Keeper, and OTel Collector configuration files:

```bash
aws s3 mb s3://my-signoz-config-bucket --region us-east-1
```

#### 5. Cloud Map Namespace (Service Connect)

Create a Cloud Map HTTP namespace for ECS Service Connect:

```bash
aws servicediscovery create-http-namespace \
  --name signoz \
  --region us-east-1
```

This namespace enables DNS-based service discovery between ECS services (e.g., `telemetrystore`, `metastore`).

#### 6. CloudWatch Log Group (optional)

Log groups are auto-created by the task definitions (`/ecs/<name>`), but you can pre-create one:

```bash
aws logs create-log-group \
  --log-group-name /ecs/signoz \
  --region us-east-1
```

## Configuration

### Annotations

The following annotations are used during forge to generate task definitions:

| Annotation | Required | Description |
|---|---|---|
| `foundry.signoz.io/ecs-region` | Yes | AWS region (used in CloudWatch log config) |
| `foundry.signoz.io/ecs-task-execution-role-arn` | Yes | IAM role ARN for ECS task execution |
| `foundry.signoz.io/ecs-task-role-arn` | Yes | IAM role ARN for task containers |
| `foundry.signoz.io/ecs-config-bucket` | Yes | S3 bucket name for config files (used in config-fetcher init containers) |

### Example casting.yaml

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-task-execution-role-arn: arn:aws:iam::123456789012:role/ecsTaskExecutionRole
    foundry.signoz.io/ecs-task-role-arn: arn:aws:iam::123456789012:role/ecsTaskRole
    foundry.signoz.io/ecs-config-bucket: my-signoz-config-bucket
spec:
  deployment:
    flavor: ec2
    platform: ecs
```

## Deployment

### Step 1. Generate deployment files (Forge)

```bash
foundryctl forge -f casting.yaml
```

This generates per-component task definitions and config files under `pours/deployment/`:

```
pours/deployment/
├── telemetrykeeper/
│   └── clickhousekeeper/
│       ├── task-definition.json
│       └── keeper-0.yaml
├── telemetrystore/
│   └── clickhouse/
│       ├── task-definition.json
│       ├── config.yaml
│       └── functions.yaml
├── telemetrystore-migrator/
│   └── task-definition.json
├── metastore/
│   └── postgres/
│       └── task-definition.json
├── signoz/
│   └── task-definition.json
└── ingester/
    ├── task-definition.json
    ├── ingester.yaml
    └── opamp.yaml
```

### Step 2. Upload config files to S3

Components with config files (telemetrykeeper, telemetrystore, ingester) have a `config-fetcher` init container that pulls configs from S3 at startup. Upload them before creating services:

```bash
BUCKET=my-signoz-config-bucket
NAME=signoz
REGION=us-east-1

# TelemetryKeeper configs
aws s3 sync pours/deployment/telemetrykeeper/clickhousekeeper/ \
  s3://${BUCKET}/${NAME}/telemetrykeeper/clickhousekeeper/ \
  --exclude "task-definition.json" \
  --region ${REGION}

# TelemetryStore configs
aws s3 sync pours/deployment/telemetrystore/clickhouse/ \
  s3://${BUCKET}/${NAME}/telemetrystore/clickhouse/ \
  --exclude "task-definition.json" \
  --region ${REGION}

# Ingester configs
aws s3 sync pours/deployment/ingester/ \
  s3://${BUCKET}/${NAME}/ingester/ \
  --exclude "task-definition.json" \
  --region ${REGION}
```

Expected S3 layout:

```
s3://my-signoz-config-bucket/signoz/
├── telemetrykeeper/clickhousekeeper/
│   └── keeper-0.yaml
├── telemetrystore/clickhouse/
│   ├── config.yaml
│   └── functions.yaml
└── ingester/
    ├── ingester.yaml
    └── opamp.yaml
```

### Step 3. Register task definitions

Register each generated task definition with ECS:

```bash
REGION=us-east-1

TASK_DEFS=(
  pours/deployment/telemetrykeeper/clickhousekeeper/task-definition.json
  pours/deployment/telemetrystore/clickhouse/task-definition.json
  pours/deployment/telemetrystore-migrator/task-definition.json
  pours/deployment/metastore/postgres/task-definition.json
  pours/deployment/signoz/task-definition.json
  pours/deployment/ingester/task-definition.json
)

for TASK_DEF in "${TASK_DEFS[@]}"; do
  if [ -f "$TASK_DEF" ]; then
    echo "Registering ${TASK_DEF}..."
    aws ecs register-task-definition \
      --cli-input-json "file://${TASK_DEF}" \
      --region ${REGION} \
      --output json
  fi
done
```

Note the task definition ARNs from the output : you'll need them for creating services.

### Step 4. Create ECS services

Create services in dependency order. Each service uses Service Connect for inter-service DNS discovery.

Set your environment variables:

```bash
CLUSTER=my-ecs-cluster
REGION=us-east-1
NAME=signoz
SUBNETS="subnet-abc123,subnet-def456"
SECURITY_GROUPS="sg-abc123"
NAMESPACE=signoz  # Cloud Map namespace
```

**TelemetryKeeper:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-telemetrykeeper \
  --task-definition ${NAME}-telemetrykeeper \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{
    "enabled": true,
    "namespace": "'${NAMESPACE}'",
    "services": [
      {"portName": "keeper-client", "discoveryName": "telemetrykeeper", "clientAliases": [{"port": 9181, "dnsName": "telemetrykeeper"}]},
      {"portName": "keeper-raft", "discoveryName": "telemetrykeeper", "clientAliases": [{"port": 9234, "dnsName": "telemetrykeeper"}]}
    ]
  }' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

**TelemetryStore:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-telemetrystore \
  --task-definition ${NAME}-telemetrystore \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{
    "enabled": true,
    "namespace": "'${NAMESPACE}'",
    "services": [
      {"portName": "clickhouse-native", "discoveryName": "telemetrystore", "clientAliases": [{"port": 9000, "dnsName": "telemetrystore"}]},
      {"portName": "clickhouse-http", "discoveryName": "telemetrystore", "clientAliases": [{"port": 8123, "dnsName": "telemetrystore"}]}
    ]
  }' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

**TelemetryStore Migrator:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-telemetrystore-migrator \
  --task-definition ${NAME}-telemetrystore-migrator \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{"enabled": true, "namespace": "'${NAMESPACE}'"}' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

**MetaStore:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-metastore \
  --task-definition ${NAME}-metastore \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{
    "enabled": true,
    "namespace": "'${NAMESPACE}'",
    "services": [
      {"portName": "postgres", "discoveryName": "metastore", "clientAliases": [{"port": 5432, "dnsName": "metastore"}]}
    ]
  }' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

**SigNoz:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-signoz \
  --task-definition ${NAME}-signoz \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{
    "enabled": true,
    "namespace": "'${NAMESPACE}'",
    "services": [
      {"portName": "signoz-http", "discoveryName": "signoz", "clientAliases": [{"port": 8080, "dnsName": "signoz"}]},
      {"portName": "signoz-opamp", "discoveryName": "signoz", "clientAliases": [{"port": 4320, "dnsName": "signoz"}]}
    ]
  }' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

**Ingester:**

```bash
aws ecs create-service \
  --cluster ${CLUSTER} \
  --service-name ${NAME}-ingester \
  --task-definition ${NAME}-ingester \
  --desired-count 1 \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNETS}],securityGroups=[${SECURITY_GROUPS}],assignPublicIp=DISABLED}" \
  --service-connect-configuration '{
    "enabled": true,
    "namespace": "'${NAMESPACE}'",
    "services": [
      {"portName": "otel-grpc", "discoveryName": "ingester", "clientAliases": [{"port": 4317, "dnsName": "ingester"}]},
      {"portName": "otel-http", "discoveryName": "ingester", "clientAliases": [{"port": 4318, "dnsName": "ingester"}]}
    ]
  }' \
  --deployment-configuration "deploymentCircuitBreaker={enable=true,rollback=true}" \
  --enable-execute-command \
  --launch-type EC2 \
  --region ${REGION}
```

> If using a capacity provider instead of `--launch-type EC2`, replace that flag with:
> `--capacity-provider-strategy capacityProvider=my-capacity-provider,weight=1,base=0`

### Updating an existing deployment

After re-running `foundryctl forge` and re-uploading configs to S3, update services with:

```bash
aws ecs update-service \
  --cluster ${CLUSTER} \
  --service ${NAME}-<component> \
  --task-definition ${NAME}-<component> \
  --region ${REGION}
```