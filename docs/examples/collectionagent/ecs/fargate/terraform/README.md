# ECS Fargate Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `fargate` |
| **Flavor** | `terraform` |

## Overview

Runs a SigNoz Collection Agent as a sidecar container inside an application's own ECS task. A Fargate task has no container instance, so there is nothing for a daemon agent to run on; `spec.collector.kind` must be `sidecar` here, and `kind: agent` is refused at forge.

ECS has no admission hook, so nothing foundry runs can add a container to a task definition foundry does not own. The pour is a Terraform module you import from your own Terraform, and your task definition takes its containers, its execution role and its log configuration from the module's outputs.

The collector runs the OpenTelemetry Collector and exports to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud. It collects:

- OTLP traces, metrics and logs from the application containers, on the task's own `localhost`
- The stdout and stderr of the application containers that opt in to its log configuration
- Task CPU, memory, network and storage metrics from the ECS task metadata endpoint

The collector is the task's FireLens container: ECS validates the router type, not the image, so it mounts the fluent socket into the collector and the collector reads the task's stdout itself, with no fluent-bit router in the task. If ECS ever validates the image too, a fluent-bit router returns as an opt-in.

Foundry generates Terraform. It does not create the cluster, the service or your task definition.

## Prerequisites

- An ECS cluster and a task definition of your own, managed by your own Terraform
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
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

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-task-execution-role-arn` | Task execution role of the task definition the collector joins. Optional; created when absent. |

The ECS agent, not the task, resolves the collector's config secret, so the read belongs to the execution role. State the ARN and the module adopts that role and attaches the one read to it. Leave it out and the module creates `signoz-collectionagent-iam-exec` with the ECS task execution policy, and `execution_role_arn` hands it back for your task definition to use.

The module has no provider and touches no cluster and no task role, so the execution role is the only annotation it reads. For the full annotation table, see the [casting file reference](../../../../../reference/casting-file.md).

### Point the collector at your SigNoz

Set the endpoint through `spec.collector.spec.env`, which becomes the container's environment. For SigNoz Cloud or Self-Hosted Enterprise, add the [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/) as an exporter header through `spec.collector.spec.config.data`:

```yaml
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

`spec.collector.spec.env` values land in the container definition in plain text. For a Self-Hosted Community installation the OTLP HTTP ingest is port `4318` on the SigNoz host and there is no ingestion key.

## Forge

```bash
foundryctl forge -f casting.yaml
```

`foundryctl cast` applies nothing, because the task definition is yours. Import the module as below and apply it with your own `terraform apply`.

## Generated output

```text
pours/collectionagent/collector/sidecar/
  versions.tf.json          # required terraform and provider versions
  main.tf.json              # the parameter holding the config, its read policy, the execution role, the collector
  outputs.tf.json           # container_definitions, log_configuration, execution_role_arn
  sidecar.yaml              # the collector config, uploaded from here
```

The module declares no variables. Everything the container needs is resolved at forge from the casting: the image from `spec.collector.spec.image`, the environment from `spec.collector.spec.env`, the config from `spec.collector.spec.config.data`. To change what the module renders, use [patches](../../../../../concepts/patches.md).

The module holds no provider, no backend and no state of its own. The root that imports it owns all three, including the region, so the collector is planned, applied and destroyed with the application it runs beside.

| Output | Meaning |
| --- | --- |
| `container_definitions` | The collector, to concat into your task definition's `container_definitions` |
| `log_configuration` | The `awsfirelens` log configuration. Applies to the application containers that opt in; without it a container's stdout is not collected |
| `execution_role_arn` | The role your task definition states as its execution role, adopted when you state one and created otherwise |

### Consume the module

```hcl
module "signoz_collector_sidecar" {
  source = "./pours/collectionagent/collector/sidecar"
}

locals {
  containers = [
    {
      name      = "app"
      image     = "<your application image>"
      essential = true

      logConfiguration = module.signoz_collector_sidecar.log_configuration
    },
  ]
}

resource "aws_ecs_task_definition" "app" {
  family                   = "app"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = 1024
  memory                   = 2048
  execution_role_arn       = module.signoz_collector_sidecar.execution_role_arn

  container_definitions = jsonencode(concat(module.signoz_collector_sidecar.container_definitions, local.containers))
}
```

`logConfiguration` is per container and opt-in. Leave it off a container and that container keeps its own stdout handling; its telemetry still reaches the collector over OTLP.

### Configuration delivery

The config is delivered through [SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html): the module uploads `sidecar.yaml` as the parameter `signoz-collectionagent-ssm-collector-sidecar`, and the collector reads it as a secret environment variable. The secret names the parameter version, so a changed config is a changed container definition and a new task revision, and a running task keeps the config it was registered with.

### Sizing

The collector reserves 256 CPU units and 256 MiB. A reservation is a floor for placement, not a cap. Fargate bills the task's CPU and memory tier, so the collector costs nothing extra until what it reserves pushes the task into the next tier up.

When it does, drop the `logConfiguration` block and have the application send its logs over OTLP instead, which removes the log path without removing the collector. One collector that every task exports to, rather than one inside every task, is what the `deployment` collector kind will offer on ECS once it lands.

The collector is not essential and carries a restart policy, so it never takes the task down and comes back on its own. Do not give it a health check: its image carries no shell, so the check fails and the task stays UNHEALTHY while the collector runs.

One task shares one network namespace, so these ports are the application containers' own `localhost`:

| Port | Bound by |
| --- | --- |
| 4317 | OTLP intake, gRPC |
| 4318 | OTLP intake, HTTP |
| 13133 | the collector's health check endpoint |

Container stdout does not arrive on a port. ECS bind-mounts the fluent socket into the collector, and the log driver writes to it.

`spec.collector.spec.cluster.replicas` is refused for anything but `1`. The count belongs to the application's own service.

## After deployment

Point [instrumented applications](https://signoz.io/docs/instrumentation/) at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP). In SigNoz, the task metrics appear under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/).

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones, and the merged result is what the parameter carries. For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md).
