# Docker Swarm Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `docker` |
| **Flavor** | `swarm` |

## Overview

Deploys a SigNoz [Docker Swarm Collection Agent](https://signoz.io/docs/opentelemetry-collection-agents/docker-swarm/overview/) as a global Swarm service: one agent runs on every node, collecting that node's telemetry and exporting it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

- Container metrics from the Docker Engine API through the `docker_stats` receiver
- Host metrics from the mounted host filesystem through the `hostmetrics` receiver
- Container logs from the Docker log files through the `filelog` receiver
- OTLP intake for your applications on ports `4317` (gRPC) and `4318` (HTTP), published on every node

Swarm has no host networking, so the service sets `hostname: "{{.Node.Hostname}}"`: the `host.name` resource attribute on all telemetry is the real node's hostname, and each node appears as itself in SigNoz. Container metrics carry the Swarm labels as `docker.stack.name`, `docker.service.name`, `docker.task.name`, and `docker.node.id`.

## Prerequisites

- Docker Engine 25.0 or newer on every node (the agent speaks Docker API 1.44)
- Docker Swarm initialized (`docker swarm init`)
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

The default casting (this directory's `casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    flavor: swarm
    mode: docker
```

The generated collector config follows the [Docker Swarm Collection Agent configuration guide](https://signoz.io/docs/opentelemetry-collection-agents/docker-swarm/configure/) and exports to `${env:SIGNOZ_INGESTION_ENDPOINT}`.

### Point the agent at your SigNoz

Set the endpoint and environment through `spec.collector.spec.env`, which becomes the service's environment:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: swarm
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "http://<signoz-host>:4318"
        OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"
```

- For a Self-Hosted Community installation the OTLP HTTP ingest is port `4318` on the SigNoz host. Community has no ingestion key, so the endpoint is all the agent needs (see [Cloud to Self-Hosted](https://signoz.io/docs/ingestion/cloud-vs-self-hosted/#cloud-to-self-hosted)).
- `OTEL_RESOURCE_ATTRIBUTES` sets the `deployment.environment` resource attribute on everything the agent collects.
- `docker stack deploy` does not interpolate variables from the shell or an `.env` file; values in the casting are written literally into the stack file at forge time.

### SigNoz Cloud or Self-Hosted Enterprise

Both authenticate ingestion with an [ingestion key](https://signoz.io/docs/ingestion/signoz-cloud/keys/). For SigNoz Cloud, set the [endpoint for your region](https://signoz.io/docs/ingestion/signoz-cloud/overview/#endpoint) (`us`, `eu`, `in`); for Self-Hosted Enterprise, use your deployment's ingestion endpoint. Add the key as an exporter header through `spec.collector.spec.config.data`:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: swarm
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

The casting is the single input; treat it as sensitive once the key is in it.

## Deploy

Point the endpoint in the casting at your SigNoz, then, on a manager node:

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Validate prerequisites
foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Deploy the stack
docker stack deploy -c pours/collectionagent/compose.yaml signoz
```

## Generated output

```text
pours/collectionagent/
  compose.yaml
  collector/
    agent/
      agent.yaml
```

## After deployment

```bash
# One task per node, all Running
docker service ps signoz_collector-agent

# View agent logs; look for "Everything is ready"
docker service logs signoz_collector-agent

# Remove the stack
docker stack rm signoz
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` on the same node, or the node's IP from other machines. To reach the agent by service name, attach it to your application's overlay network with a [patch](../../../../concepts/patches.md).

In SigNoz, each Swarm node appears under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/) by its hostname, and container metrics filter by `docker.stack.name` and `docker.service.name` under [Docker container metrics](https://signoz.io/docs/metrics-management/docker-container-metrics/).

> [!IMPORTANT]
> The collector config is delivered as an immutable Swarm config, and `docker stack deploy` cannot update an in-use config's content. To apply collector config changes, remove and redeploy the stack:
>
> ```bash
> docker stack rm signoz && foundryctl cast -f casting.yaml
> ```
>
> Changes to `spec.env` alone update in place with a plain re-`cast`; only `spec.collector.spec.config` changes need the remove-and-redeploy.

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones. For changes to the generated `compose.yaml` itself (overlay networks, placement constraints, resource limits), use [patches](../../../../concepts/patches.md).
