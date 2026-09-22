# ECS EC2 Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `ecs` |
| **Mode** | `ec2` |
| **Flavor** | `terraform` |

## Overview

The Collection Agent runs as an ECS daemon service on an existing cluster: one task on every EC2 container instance registered to it. `foundryctl` generates the Terraform and applies it. The container runs the OpenTelemetry Collector and sends telemetry to SigNoz Cloud, Self-Hosted Enterprise or Self-Hosted Community.

It collects:

- OTLP traces, metrics and logs from the tasks on the instance, on the instance's own `localhost:4317` (gRPC) and `localhost:4318` (HTTP)
- The stdout and stderr of every container on the instance, read from the Docker `json-file` logs through the `filelog` receiver and labelled with the ECS cluster, task ARN, task family and container name
- Per-container CPU, memory, network and block IO metrics from the Docker Engine API through the `docker_stats` receiver
- Instance CPU, memory, disk, filesystem, network, paging and process metrics through the `hostmetrics` receiver
- The instance's own `/var/log/messages`, `/var/log/secure` and ECS agent logs through a second `filelog` receiver
- EC2 instance identity as resource attributes through the `resourcedetection` processor

One collector runs per container instance, whatever the number of tasks on it. The task uses the `host` network mode and mounts the Docker socket, the Docker log directory and the host filesystem read-only. The collector runs as root, because Docker writes its log files readable only by root. The collector filters its own task family out of the log stream, so it does not ship its own output back to itself.

Containers using the `awslogs` log driver write nothing to the instance, so their logs are not collected. `foundryctl` does not create the cluster.

## Prerequisites

- [foundryctl](../../../../../getting-started.md)
- An ECS cluster with registered EC2 container instances. Fargate has no host to run a daemon on.
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.4 or newer
- AWS credentials in the environment Terraform runs in, with permission to create AppConfig applications, IAM roles, ECS task definitions and ECS services
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

This directory's `agent/casting.yaml`:

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

`foundry.signoz.io/ecs-region` is the AWS region holding the cluster, and is required.

`foundry.signoz.io/ecs-cluster-arn` is the ARN of the cluster the daemon service runs on, and is required.

`foundry.signoz.io/ecs-task-role-arn` is the role the agent task assumes to read its config; state yours to use it, omit it and Terraform creates `signoz-collectionagent-iam-task`.

`foundry.signoz.io/ecs-task-execution-role-arn` is the role the ECS agent assumes to pull images and start the task; state yours to use it, omit it and Terraform creates `signoz-collectionagent-iam-exec`.

The collector config is delivered through [AWS AppConfig](https://docs.aws.amazon.com/appconfig/). A second container in the task fetches the configuration profile and writes it to `/conf/agent.yaml`, and the collector starts once that container is healthy.

### SigNoz Cloud or Self-Hosted Enterprise

Both authenticate ingestion with an [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/). For SigNoz Cloud, use the [endpoint for your region](https://signoz.io/docs/ingestion/signoz-cloud/overview/#endpoint); for Self-Hosted Enterprise, use your deployment's ingestion endpoint. Send the key as an exporter header:

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
        SIGNOZ_INGESTION_KEY: "<your-ingestion-key>"
      config:
        data:
          collector/agent/agent.yaml: |
            exporters:
              otlphttp/signoz:
                headers:
                  signoz-ingestion-key: ${env:SIGNOZ_INGESTION_KEY}
```

Values under `spec.collector.spec.env` are visible in the task definition.

### Self-Hosted Community

Set `SIGNOZ_INGESTION_ENDPOINT` to `http://<signoz-host>:4318` and set no key. The SigNoz UI is on port 8080 of the same host.

## Deploy

```bash
foundryctl gauge -f casting.yaml
foundryctl forge -f casting.yaml
foundryctl cast -f casting.yaml
```

`gauge` checks that Terraform is installed and usable. `forge` writes the Terraform root under `pours/`. `cast` runs `terraform init`, `terraform plan -out=tfplan` and `terraform apply tfplan` in that root.

Then point [instrumented applications](https://signoz.io/docs/instrumentation/) on the instance at `http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP). A task using the `host` network mode reaches the agent there. A task using `awsvpc` has its own loopback and must address the instance's private IP instead.

## Generated output

```text
pours/collectionagent/collector/agent/
  versions.tf.json          # required Terraform and provider versions
  providers.tf.json         # the aws provider, in var.aws_region
  backend.tf.json           # local state, in this directory
  variables.tf.json         # every identifier the root declares, each validated
  terraform.tfvars.json     # the values the casting resolved
  main.tf.json              # the AppConfig application, environment and strategy, and the two roles
  collector.tf.json         # the configuration profile and deployment, the task definition, the daemon service
  agent.yaml                # the collector config, uploaded from here
```

Terraform keeps `terraform.tfstate` in that directory. Keep it: `terraform destroy` needs it to know what to remove. Forging into a different location leaves the state behind, so carry the file across, or hold the state remotely by patching `backend.tf.json`.

## After deployment

```bash
# One task per container instance
aws ecs describe-services --cluster <cluster> --services signoz-collector-agent

# The tasks themselves
aws ecs list-tasks --cluster <cluster> --service-name signoz-collector-agent

# Agent health, from the instance
curl -fsS localhost:13133/healthz && echo " OK"
```

To change what the collector does, edit `casting.yaml`, forge again, and cast. A changed config registers a new task revision and the daemon service rolls it out. The service has a deployment circuit breaker, so a revision that never becomes healthy rolls back instead of reaching every instance.

In SigNoz, the instances appear under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/), and the container metrics feed the [Instance Metrics - ECS](https://github.com/SigNoz/dashboards/tree/main/ecs-infra-metrics) dashboard.

To remove the daemon service, its task definition and its AppConfig application:

```bash
cd pours/collectionagent/collector/agent && terraform destroy
```

## Customization

Override any collector setting through `spec.collector.spec.config.data`, keyed by the generated file's path. Your keys win over the generated ones, and the merged result is what AppConfig delivers. For example, to flush batches sooner:

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
      config:
        data:
          collector/agent/agent.yaml: |
            processors:
              batch:
                timeout: 5s
```

For changes to the generated Terraform itself, use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Meaning |
| --- | --- |
| `foundry.signoz.io/ecs-region` | AWS region holding the cluster. Required. |
| `foundry.signoz.io/ecs-cluster-arn` | ARN of the ECS cluster to run the agent on. Required. |
| `foundry.signoz.io/ecs-task-role-arn` | IAM role ARN assumed by the agent task; needs read access to AWS AppConfig. Created when absent. |
| `foundry.signoz.io/ecs-task-execution-role-arn` | IAM role ARN the ECS agent assumes to pull images and start tasks. Created when absent. |
