# Docker Compose Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `docker` |
| **Flavor** | `compose` |

## Overview

Deploys a SigNoz [Docker Collection Agent](https://signoz.io/docs/opentelemetry-collection-agents/docker/overview/) on a Docker host as a single Compose service. The agent runs the OpenTelemetry Collector, collects the host's telemetry, and exports it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

- Container metrics from the Docker Engine API through the `docker_stats` receiver
- Host metrics from the mounted host filesystem through the `hostmetrics` receiver
- Container logs from the Docker log files through the `filelog` receiver
- OTLP intake for your applications on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

The agent runs with host networking, so the `host.name` resource attribute on all telemetry is the real host's name and applications on the host reach the agent on localhost.

## Prerequisites

- Docker Engine 25.0 or newer (the agent speaks Docker API 1.44)
- Docker Compose v2
- A running SigNoz to receive the telemetry: [Self-Hosted Community](../../../docker/compose/README.md), Self-Hosted Enterprise, or [SigNoz Cloud](https://signoz.io/teams/)

## Configuration

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "http://<signoz-host>:4318"
        OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"
```

- `spec.collector.spec.env` becomes the container's environment. Point `SIGNOZ_INGESTION_ENDPOINT` at your SigNoz OTLP HTTP ingest; for a Self-Hosted Community installation that is port `4318` on the SigNoz host. Community has no ingestion key, so the endpoint is all the agent needs (see [Cloud to Self-Hosted](https://signoz.io/docs/ingestion/cloud-vs-self-hosted/#cloud-to-self-hosted)).
- `OTEL_RESOURCE_ATTRIBUTES` sets the `deployment.environment` resource attribute on everything the agent collects. Adjust it per host.

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
    flavor: compose
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

The casting is the single input; treat it as sensitive once the key is in it. `spec.collector.spec.config.data` merges into the generated collector config, so the same channel overrides any other collector setting.

## Deploy

Point the endpoint in the casting at your SigNoz, then:

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Validate prerequisites
foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Start the agent
cd pours/collectionagent && docker compose up -d
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
# Check the agent is healthy
curl -fsS localhost:13133/healthz && echo " OK"

# View agent logs
docker compose -f pours/collectionagent/compose.yaml logs -f

# Stop the agent
docker compose -f pours/collectionagent/compose.yaml down
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) on the host at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP). Containers on the default bridge network reach it at the bridge gateway, typically `172.17.0.1`.

In SigNoz, the host appears under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/) and per-container metrics under [Docker container metrics](https://signoz.io/docs/metrics-management/docker-container-metrics/).

> [!IMPORTANT]
> **Upgrading:** re-running `cast` regenerates the config files, but Docker Compose does not restart containers when only mounted file contents change. Recreate the agent so the new config takes effect:
>
> ```bash
> docker compose -f pours/collectionagent/compose.yaml up -d --force-recreate
> ```

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones. For changes to the generated `compose.yaml` itself (extra mounts, capabilities, resource limits), use [patches](../../../../concepts/patches.md).
