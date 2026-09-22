# ECS Fargate Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `fargate` |
| **Flavor** | `terraform` |

## Overview

Runs a SigNoz Collection Agent as a sidecar container inside your own ECS task on Fargate. The collector runs the OpenTelemetry Collector, collects the task's telemetry, and exports it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

Foundry generates a Terraform module and deploys nothing itself. You import the module from the Terraform that owns your task definition and apply it yourself; `foundryctl cast` only prints where the module is.

It collects:

- OTLP traces, metrics and logs from your application containers, on the task's own `localhost:4317` (gRPC) and `localhost:4318` (HTTP)
- The stdout and stderr of the application containers that opt in to its log configuration
- Eight task-level metrics from the ECS task metadata endpoint: `ecs.task.cpu.*`, `ecs.task.memory.*`, `ecs.task.network.*` and `ecs.task.storage.*`

Every container in a Fargate task shares one network namespace, so your applications reach the collector on localhost. On EC2 tasks with `bridge` networking they do not, and this casting targets Fargate.

## Prerequisites

- [foundryctl](../../../../../getting-started.md) to generate the files
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- A task definition of your own, managed by your own Terraform
- AWS credentials in the environment Terraform runs in, with permission to create SSM parameters and IAM role policies
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

The default casting (this directory's `sidecar/casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/ecs-task-execution-role-arn: arn:aws:iam::123456789012:role/app-task-exec
spec:
  deployment:
    platform: ecs
    mode: fargate
    flavor: terraform
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
```

`spec.collector.kind` is `sidecar`, and any other kind is refused at forge: a Fargate task has no host for a daemon agent to run on. `spec.collector.spec.cluster.replicas` stays `1`, one collector per task, and the count belongs to your own service.

State `foundry.signoz.io/ecs-task-execution-role-arn` to use your task's existing execution role, and the module adopts it. Omit it and the module creates one and hands it back for your task definition to use.

### Point the agent at your SigNoz

`SIGNOZ_INGESTION_ENDPOINT` and everything else under `spec.collector.spec.env` becomes the container's environment.

- For a Self-Hosted Community installation the OTLP HTTP ingest is port `4318` on the SigNoz host, `http://<signoz-host>:4318`. Community has no ingestion key, so the endpoint is all the collector needs (see [Cloud to Self-Hosted](https://signoz.io/docs/ingestion/cloud-vs-self-hosted/#cloud-to-self-hosted)).
- `OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"` sets the `deployment.environment` resource attribute on everything the collector sends. Adjust it per task.

### SigNoz Cloud or Self-Hosted Enterprise

Both authenticate ingestion with an [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/). For SigNoz Cloud, set the [endpoint for your region](https://signoz.io/docs/ingestion/signoz-cloud/overview/#endpoint) (`us`, `eu`, `in`); for Self-Hosted Enterprise, use your deployment's ingestion endpoint. Add the key as an exporter header through `spec.collector.spec.config.data`:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: fargate
    flavor: terraform
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
        SIGNOZ_INGESTION_KEY: "<your-ingestion-key>"
        OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"
      config:
        data:
          collector/sidecar/sidecar.yaml: |
            exporters:
              otlphttp/signoz:
                headers:
                  signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
```

The casting is the single input; treat it as sensitive once the key is in it. `spec.collector.spec.env` values land in the container definition in plain text.

## Deploy

```bash
# Validate prerequisites
foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml
```

Then import the module from the Terraform that owns your task definition. It takes no inputs: the task definition states the module's execution role, concatenates its container into `container_definitions`, and every application container whose stdout should be collected takes the module's log configuration.

```hcl
module "signoz_collector_sidecar" {
  source = "./pours/collectionagent/collector/sidecar"
}

resource "aws_ecs_task_definition" "app" {
  family                   = "app"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 1024
  memory                   = 2048
  execution_role_arn       = module.signoz_collector_sidecar.execution_role_arn

  container_definitions = jsonencode(concat(module.signoz_collector_sidecar.container_definitions, [
    {
      name      = "app"
      image     = "<your application image>"
      essential = true

      logConfiguration = module.signoz_collector_sidecar.log_configuration
    },
  ]))
}
```

Apply it with your own Terraform, `terraform init && terraform apply`, and point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

`logConfiguration` is per container and opt-in. Leave it off a container and that container keeps its own log driver; its OTLP telemetry still reaches the collector. Opting a container in makes the collector the task's FireLens log router, so the task can carry no other FireLens container.

The collector is one more container in every task and Fargate bills the task's CPU and memory tier, so size the task with 256 CPU units and 256 MiB on top of what your application needs.

## Generated output

```text
pours/collectionagent/collector/sidecar/
  versions.tf.json          # required terraform and provider versions
  main.tf.json              # the collector container, the parameter holding its config, and the read policy
  outputs.tf.json           # container_definitions, log_configuration, execution_role_arn
  sidecar.yaml              # the collector config, uploaded from here
```

The module declares no variables and carries no provider, no backend and no state of its own. The Terraform that imports it owns all three, so the collector is planned, applied and destroyed with the application it runs beside.

## After deployment

```bash
# The collector runs beside your application containers
aws ecs describe-tasks --cluster <cluster> --tasks <task-arn> \
  --query "tasks[].containers[?name=='signoz-collector-sidecar'].lastStatus"
```

The collector is not essential and restarts on its own, so a crashed collector never takes your application down, but telemetry is missing until it comes back.

The collector's own stdout is not shipped anywhere, because it is the task's log router and carries no log configuration of its own. To read it in CloudWatch, give it one with a patch; the log group has to exist already:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: fargate
    flavor: terraform
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
  patches:
    - target: collectionagent/collector/sidecar/main.tf.json
      operations:
        - op: add
          path: /locals/container_collector_sidecar/logConfiguration
          value:
            logDriver: awslogs
            options:
              awslogs-group: /ecs/app
              awslogs-region: us-east-1
              awslogs-stream-prefix: collector
```

To change what the collector does, edit the casting, forge again, and apply. The config is delivered as an SSM parameter whose version is part of the container definition, so a changed config registers a new task revision and your service rolls it out; a running task keeps the config it started with.

In SigNoz, your applications' traces, metrics and logs arrive as usual, and the task metrics feed the [Container Metrics - ECS](https://github.com/SigNoz/dashboards/tree/main/ecs-infra-metrics) dashboard.

## Customization

Override any collector setting through `spec.collector.spec.config.data`, keyed by the generated file's path; user keys win over generated ones, and the merged result is what the task reads. For example, to flush batches sooner:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    platform: ecs
    mode: fargate
    flavor: terraform
  collector:
    kind: sidecar
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "https://ingest.us.signoz.cloud:443"
      config:
        data:
          collector/sidecar/sidecar.yaml: |
            processors:
              batch:
                timeout: 5s
```

For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-task-execution-role-arn` | Execution role of the task definition the collector joins. Optional; created when absent. |

The other `foundry.signoz.io/ecs-*` annotations are not read here: the module touches no region, no cluster and no task role. For the full annotation table, see the [casting file reference](../../../../../reference/casting-file.md).
