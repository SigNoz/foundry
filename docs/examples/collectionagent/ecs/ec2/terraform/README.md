# ECS EC2 Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |

## Overview

Deploys a SigNoz Collection Agent onto an existing Amazon ECS cluster backed by EC2 container instances, as an ECS **daemon service**: one agent task on every instance registered to the cluster. The agent runs the OpenTelemetry Collector, collects each instance's telemetry, and exports it, along with anything the tasks on that instance send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

- Container metrics from the instance's Docker Engine API through the `docker_stats` receiver, with the ECS cluster, task ARN, family and revision carried onto every metric from the labels ECS stamps on each container
- Instance metrics from the mounted host filesystem through the `hostmetrics` receiver
- Container logs from the Docker log files through the `filelog` receiver
- Task and instance identity through the `resourcedetection` processor's `ecs` and `ec2` detectors
- OTLP intake for your tasks on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

The task uses the `host` network mode, so the agent binds its ports on the instance itself and tasks on that instance reach it on localhost.

Foundry generates Terraform; it does not create the cluster. Everything the agent needs from AWS is either named by annotation or created by the stack.

## Prerequisites

- An ECS cluster with registered EC2 container instances (the EC2 launch type; Fargate has no host to run a daemon on)
- Terraform 1.4 or newer, and AWS credentials in the environment with permission to create AppConfig applications, IAM roles, task definitions and services
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

### Annotations

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region holding the cluster. Required. |
| `foundry.signoz.io/ecs-cluster-arn` | ARN of the ECS cluster to run the agent on. Required. |
| `foundry.signoz.io/ecs-task-role-arn` | IAM role the agent task assumes. Created when absent. |
| `foundry.signoz.io/ecs-task-execution-role-arn` | IAM role the ECS agent assumes to pull images. Created when absent. |

The two roles hold no data and die with the stack, so an absent one is created under the workload's own name (`signoz-collectionagent-iam-task`, `signoz-collectionagent-iam-exec`) rather than looked up. State an ARN instead and nothing is created; that role then needs `appconfig:StartConfigurationSession` and `appconfig:GetLatestConfiguration`, which the created role gets automatically.

`metadata.name` feeds every derived name, and AWS caps them: above 29
characters the AppConfig deployment strategy name overflows its 64-character
limit, and above 39 the IAM role names overflow theirs. Nothing refuses this at
forge yet, so terraform is where it surfaces.

### Point the agent at your SigNoz

Set the endpoint and environment through `spec.collector.spec.env`, which becomes the task's container environment. For SigNoz Cloud or Self-Hosted Enterprise, add the [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/) as an exporter header through `spec.collector.spec.config.data`:

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

## Config delivery

The collector config travels through [AWS AppConfig](https://docs.aws.amazon.com/appconfig/), the ECS analog of a ConfigMap, not through a bucket:

1. Terraform creates an AppConfig application (`signoz-collectionagent-appconfig`), a `default` environment, and a configuration profile (`collector-agent`) holding the generated `agent.yaml` as a hosted configuration version.
2. An [AppConfig agent](https://docs.aws.amazon.com/appconfig/latest/userguide/appconfig-integration-containers-agent.html) sidecar in the task fetches that profile and writes it to `/conf/agent.yaml` on a task volume. The collector container waits for the sidecar to report healthy before it starts.
3. The task definition carries a `FOUNDRY_CONFIG_DIGEST` of the config. The collector reads its config once at start, so a changed config has to replace the task; the digest is what makes the revision change.

The AppConfig application carries the Kind in its name, so a CollectionAgent and an Installation of the same `metadata.name` hold their own configuration on one account.

## Giving your logs a service name

The agent reads container logs off the instance's disk, where the only identity
is a 64-character container ID. For a log line to carry the service that
produced it, **the producing task definition** has to ask Docker to write the
ECS labels into every line:

```json
"logConfiguration": {
  "logDriver": "json-file",
  "options": {
    "max-size": "10m",
    "max-file": "3",
    "labels": "com.amazonaws.ecs.task-definition-family,com.amazonaws.ecs.container-name,com.amazonaws.ecs.task-arn,com.amazonaws.ecs.cluster"
  }
}
```

ECS already stamps those four labels on every container. The `labels` option
copies their values into an `attrs` object on each line, and the agent lifts
them onto the record:

| Label | Becomes |
| --- | --- |
| `…task-definition-family` | `service.name` |
| `…container-name` | `aws.ecs.container.name` |
| `…task-arn` | `aws.ecs.task.arn` |
| `…cluster` | `aws.ecs.cluster.name` |
| container ID from the file path | `container.id` |

Two things follow from this.

- **A task without that block still reports**, but with only `container.id`. It
  is not dropped.
- **Containers using the `awslogs` driver are invisible to the agent.** That
  driver sends straight to CloudWatch and writes nothing to the instance, so
  there is no file to read and no error to see. Reading those back requires the
  CloudWatch receiver and the CloudWatch cost, which is outside this agent.

SigNoz's own components deployed by the ECS installation casting do not carry
the block yet, so their logs arrive with only a container ID for now.

## Reading the agent's own logs

Both agent containers run with `logDriver: none` by default. That is deliberate:
the agent tails `/var/lib/docker/containers`, so anything it wrote there it
would read back and re-export, and every log line it emitted would produce
another one.

To read them while testing, patch the driver to `awslogs`:

```yaml
spec:
  patches:
    - target: "collectionagent/collector.tf.json"
      operations:
        - op: replace
          path: /locals/containers_collector/1/logConfiguration
          value:
            logDriver: awslogs
            options:
              awslogs-group: /signoz/collectionagent/collector/agent
              awslogs-region: us-east-1
              awslogs-stream-prefix: ecs
```

The log group has to exist first, and the execution role already carries
`AmazonECSTaskExecutionRolePolicy`, which grants the writes.

## Deploy

```bash
foundryctl cast -f casting.yaml --yes
```

Or step by step:

```bash
# Validate prerequisites
foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Apply them
cd pours/collectionagent && terraform init && terraform apply
```

`cast` runs `terraform init`, writes a plan, then applies it. Terraform prompts for approval on its own, so foundry requires `--yes` before it will apply.

## Generated output

```text
pours/collectionagent/
  versions.tf.json          # required terraform and provider versions
  providers.tf.json         # the aws provider, in var.aws_region
  backend.tf.json           # local state, beside the pours
  variables.tf.json         # every identifier, defaulted to what the casting resolved
  terraform.tfvars.json     # the region
  main.tf.json              # AppConfig application, environment, strategy; the two roles
  collector.tf.json         # config profile and deployment, task definition, daemon service
  collector/
    agent/
      agent.yaml            # the collector config, uploaded from here
```

Terraform state lives beside the pours. Keep it: `melt` needs it to know what to remove.

## Ports on the instance

Host networking means these are the instance's own ports, not the task's:

| Port | Bound by |
| --- | --- |
| 4317 / 4318 | OTLP intake, gRPC and HTTP |
| 13133 | the collector's health check endpoint |
| 2772 | the AppConfig agent's local HTTP endpoint |

## After deployment

```bash
# One task per container instance
aws ecs describe-services --cluster <cluster> --services signoz-collector-agent

# Agent health, from the instance
curl -fsS localhost:13133/healthz && echo " OK"

# Remove the daemon service, its task definition and its AppConfig application
foundryctl melt -f casting.yaml --yes
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP) for tasks using the `host` network mode. Tasks using `awsvpc` do not share the instance's loopback and must address the instance's private IP instead.

In SigNoz, the instances appear under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/) and per-container metrics under [Docker container metrics](https://signoz.io/docs/metrics-management/docker-container-metrics/).

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones, and the merged result is what AppConfig delivers. For changes to the generated Terraform itself (task size, extra mounts, log configuration), use [patches](../../../../../concepts/patches.md).
