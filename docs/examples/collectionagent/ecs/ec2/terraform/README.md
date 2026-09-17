# ECS EC2 Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |

## Overview

Deploys a SigNoz Collection Agent onto an existing Amazon ECS cluster as a daemon service: one agent task on every EC2 container instance registered to the cluster. The agent runs the OpenTelemetry Collector and exports to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

It collects:

- Container metrics from the instance's Docker Engine API through the `docker_stats` receiver
- Instance metrics from the mounted host filesystem through the `hostmetrics` receiver
- Container logs from the Docker JSON log files, and the instance's own `/var/log` messages, secure and ECS agent logs, through the `filelog` receivers
- Instance identity through the `resourcedetection` processor's `env` and `ec2` detectors

The task uses the `host` network mode, so OTLP intake is on the instance's own `localhost:4317` (gRPC) and `localhost:4318` (HTTP). Tasks on that instance using the `host` network mode reach the agent there. Tasks using `awsvpc` have their own loopback and must address the instance's private IP instead.

Containers using the `awslogs` log driver write nothing to the instance, so the agent does not read their logs. The agent's own containers log through `json-file` on the instance, readable with `docker logs`, and the agent drops its own lines from what it ships.

Foundry generates Terraform. It does not create the cluster.

## Prerequisites

- An ECS cluster with registered EC2 container instances. Fargate has no host to run a daemon on.
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- AWS credentials in the environment Terraform runs in, with permission to create AppConfig applications, IAM roles, task definitions and services
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

The default casting (this directory's `casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
    foundry.signoz.io/ecs-cluster-arn: arn:aws:ecs:us-east-1:123456789012:cluster/signoz
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
```

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region holding the cluster. Required. |
| `foundry.signoz.io/ecs-cluster-arn` | ARN of the ECS cluster to run the agent on. Required. |
| `foundry.signoz.io/ecs-task-role-arn` | IAM role the agent task assumes. Created when absent. |
| `foundry.signoz.io/ecs-task-execution-role-arn` | IAM role the ECS agent assumes to pull images. Created when absent. |

Terraform validates the region and the cluster ARN at plan. State a role ARN and nothing is created; that role needs `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration`. Leave it out and the role is created as `signoz-collectionagent-iam-task` or `signoz-collectionagent-iam-exec`. For the full annotation table, see the [casting file reference](../../../../../reference/casting-file.md).

### Point the agent at your SigNoz

Set the endpoint through `spec.collector.spec.env`, which becomes the task's container environment. For SigNoz Cloud or Self-Hosted Enterprise, add the [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/) as an exporter header through `spec.collector.spec.config.data`:

```yaml
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
        SIGNOZ_INGESTION_KEY: "<your-ingestion-key>"
        OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"
      config:
        data:
          collector/agent/agent.yaml: |
            exporters:
              otlphttp/signoz:
                headers:
                  signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
```

`spec.collector.spec.env` values land in the task definition in plain text. For a Self-Hosted Community installation the OTLP HTTP ingest is port `4318` on the SigNoz host and there is no ingestion key.

The collector config is delivered through [AWS AppConfig](https://docs.aws.amazon.com/appconfig/): a sidecar in the task fetches it and writes it to `/conf/agent.yaml`, and the collector starts once the sidecar is healthy. A changed config replaces the task.

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
cd pours/collectionagent
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

`cast` runs the same three terraform steps.

## Generated output

```text
pours/collectionagent/
  versions.tf.json          # required terraform and provider versions
  providers.tf.json         # the aws provider, in var.aws_region
  backend.tf.json           # local state, beside the pours
  variables.tf.json         # every identifier the root declares, each validated
  terraform.tfvars.json     # the values the casting resolved
  main.tf.json              # AppConfig application, environment, strategy; the two roles
  collector.tf.json         # config profile and deployment, task definition, daemon service
  collector/
    agent/
      agent.yaml            # the collector config, uploaded from here
```

Terraform keeps its state file beside the pours. Keep it: `terraform destroy` needs it to know what to remove. To hold state remotely instead, patch `backend.tf.json`.

Host networking means these are the instance's own ports:

| Port | Bound by |
| --- | --- |
| 4317 | OTLP intake, gRPC |
| 4318 | OTLP intake, HTTP |
| 13133 | the collector's health check endpoint |

## After deployment

```bash
# One task per container instance
aws ecs describe-services --cluster <cluster> --services signoz-collector-agent

# Agent health, from the instance
curl -fsS localhost:13133/healthz && echo " OK"

# Remove the daemon service, its task definition and its AppConfig application
cd pours/collectionagent && terraform destroy
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP). In SigNoz, the instances appear under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/) and per-container metrics under [Docker container metrics](https://signoz.io/docs/metrics-management/docker-container-metrics/).

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones, and the merged result is what AppConfig delivers. For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md). For example, to hold the state in S3:

```yaml
spec:
  patches:
    - target: "collectionagent/backend.tf.json"
      operations:
        - op: replace
          path: /terraform/backend
          value:
            s3:
              bucket: my-terraform-state
              key: signoz/collectionagent.tfstate
              region: us-east-1
```

## Sidecar

The daemon above needs an EC2 container instance to run on. A task on Fargate has none, and an application that wants a collector of its own does not need one: the collector runs as a sidecar container inside the application's own task, on `localhost` with everything else in it.

ECS has no admission hook, so nothing foundry runs can add a container to a task definition foundry does not own. The sidecar pour is a Terraform module you import from your own Terraform, and your task definition derives the collector container from its outputs.

Set `spec.collector.kind` to `sidecar` (this directory's `sidecar/casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-region: us-east-1
spec:
  deployment:
    platform: ecs
    mode: ec2
    flavor: terraform
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
```

The module holds no provider, no backend and no state of its own. The root that imports it owns all three, including the region, so the collector is planned, applied and destroyed with the application it runs beside.

### Generated output

```text
pours/collectionagent/
  versions.tf.json          # required terraform and provider versions
  variables.tf.json         # the one identifier the module cannot derive
  main.tf.json              # the parameter holding the config, the read policy, the container definition
  outputs.tf.json           # container_definition, log_configuration, parameter_arn
  collector/
    sidecar/
      sidecar.yaml          # the collector config, uploaded from here
```

| Input | Meaning |
| --- | --- |
| `execution_role_name` | Name of the task execution role of the task definition the collector joins. The module attaches a policy to it that reads the configuration parameter. |

| Output | Meaning |
| --- | --- |
| `container_definition` | The collector container, to concat into `container_definitions` |
| `log_configuration` | The `awsfirelens` log configuration the application containers carry to ship their logs to the collector |
| `parameter_arn` | ARN of the SSM parameter holding the collector config |

Everything else the container needs is resolved at forge from the casting: the image from `spec.collector.spec.image`, the environment from `spec.collector.spec.env`, the config from `spec.collector.spec.config.data`. To change what the module renders, use [patches](../../../../../concepts/patches.md); the module declares no variable for it.

### Consume the module

```hcl
module "signoz_sidecar" {
  source = "./pours/collectionagent"

  execution_role_name = aws_iam_role.execution.name
}

resource "aws_ecs_task_definition" "app" {
  family                   = "app"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 1024
  memory                   = 2048
  execution_role_arn       = aws_iam_role.execution.arn

  container_definitions = jsonencode(concat([module.signoz_sidecar.container_definition], [
    {
      name              = "log_router"
      image             = "public.ecr.aws/aws-observability/aws-for-fluent-bit:stable"
      essential         = true
      memoryReservation = 50

      firelensConfiguration = {
        type = "fluentbit"
      }
    },
    {
      name      = "app"
      image     = "<your application image>"
      essential = true

      logConfiguration = module.signoz_sidecar.log_configuration
    },
  ]))
}
```

Apply it with your own `terraform apply`. `foundryctl cast` prints this wiring and applies nothing, because the task definition is yours.

The `log_router` container is ECS's own requirement: the `awsfirelens` log driver needs a FireLens container in the task, and it forwards what the application containers write to the collector on `127.0.0.1:8006`. Leave both out if your application sends its logs over OTLP instead.

Do not give the collector container a health check. Its image carries no shell, so the check fails and the task stays UNHEALTHY while the collector runs.

One task shares one network namespace, so these ports are the application containers' own `localhost`:

| Port | Bound by |
| --- | --- |
| 4317 | OTLP intake, gRPC |
| 4318 | OTLP intake, HTTP |
| 8006 | log intake, fluentforward |
| 13133 | the collector's health check endpoint |

The config is delivered through [SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html): the module uploads `collector/sidecar/sidecar.yaml` as the parameter `/signoz-collectionagent/collector/sidecar`, and the collector reads it as a secret environment variable. A changed config is a changed parameter version and a new task revision.

`spec.collector.spec.cluster.replicas` is refused for anything but `1`. The count belongs to the application's own service.
