# ECS Fargate Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `fargate` |
| **Flavor** | `terraform` |

## Overview

The Collection Agent runs as a sidecar container inside your own ECS task on Fargate. `foundryctl` generates a Terraform module, and you import it from the Terraform that manages your task definition. The container runs the OpenTelemetry Collector and sends telemetry to SigNoz Cloud, Self-Hosted Enterprise or Self-Hosted Community.

It collects:

- OTLP traces, metrics and logs from your containers, on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)
- The stdout and stderr of the containers that opt in, attributed to the container as `service.name`
- Task-level CPU, memory, network and storage metrics from the ECS task metadata endpoint

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- AWS credentials in the environment Terraform runs in, with permission to create SSM parameters and IAM role policies
- A task definition managed by your own Terraform
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

This directory's `sidecar/casting.yaml`:

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

The execution role annotation is optional: state it to use your task's existing execution role, omit it and the module creates one. `SIGNOZ_INGESTION_ENDPOINT` is the OTLP endpoint the collector exports to.

### SigNoz Cloud or Self-Hosted Enterprise

Both authenticate ingestion with an [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/). For SigNoz Cloud, use the [endpoint for your region](https://signoz.io/docs/ingestion/signoz-cloud/overview/#endpoint); for Self-Hosted Enterprise, use your deployment's ingestion endpoint. Send the key as an exporter header:

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
      config:
        data:
          collector/sidecar/sidecar.yaml: |
            exporters:
              otlphttp/signoz:
                headers:
                  signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
```

Values under `spec.collector.spec.env` are visible in the task definition.

### Self-Hosted Community

Set `SIGNOZ_INGESTION_ENDPOINT` to `http://<signoz-host>:4318` and set no key.

## Deploy

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
```

Add the module to the Terraform that manages your task definition. It takes no inputs:

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
      name             = "app"
      image            = "<your application image>"
      essential        = true
      logConfiguration = module.signoz_collector_sidecar.log_configuration
    },
  ]))
}
```

Apply it with `terraform init && terraform apply`, then point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

- Add 256 CPU units and 256 MiB to the task size for the collector.
- `logConfiguration` is per container and optional, and a task that uses it can carry no other FireLens log router.
- The collector is not marked essential and restarts on its own, so it never stops your application.

## Generated output

```text
pours/collectionagent/collector/sidecar/
  versions.tf.json          # required Terraform and provider versions
  main.tf.json              # the collector container, its config parameter, and the read policy
  outputs.tf.json           # container_definitions, log_configuration, execution_role_arn
  sidecar.yaml              # the collector config
```

## After deployment

To change what the collector does, edit `casting.yaml`, forge again, and apply. A changed config registers a new task revision and your service rolls it out.

The collector's own stdout is not sent anywhere. To read it in CloudWatch, give it a log configuration with a patch. The log group has to exist already:

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

The task metrics feed the [Container Metrics - ECS](https://github.com/SigNoz/dashboards/tree/main/ecs-infra-metrics) dashboard.

## Customization

Override any collector setting through `spec.collector.spec.config.data`, keyed by the generated file's path. For example, to flush batches sooner:

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
