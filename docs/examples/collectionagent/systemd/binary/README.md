# Systemd Binary Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `systemd` |
| **Flavor** | `binary` |

## Overview

Deploys a SigNoz [VM Collection Agent](https://signoz.io/docs/opentelemetry-collection-agents/vm/overview/) on a Linux host as a systemd service running the OpenTelemetry Collector Contrib binary you place yourself (no package manager). The agent collects the host's telemetry and exports it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

- Host metrics (CPU, memory, disk, filesystem, load, network, paging, process, NFS) through the `hostmetrics` receiver, running natively on the host (no container, no bind mount)
- OTLP intake for your applications on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

The collector runs directly on the host, so the `host.name` resource attribute on all telemetry is the real host's name.

## Prerequisites

- A Linux host with `systemd`
- The [OpenTelemetry Collector Contrib binary](https://github.com/open-telemetry/opentelemetry-collector-releases/releases) downloaded and placed at `/usr/local/bin/otelcol-contrib` (or wherever the `foundry.signoz.io/collector-binary-path` annotation points):

  ```bash
  ARCH=$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')
  curl -sSL -O "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.139.0/otelcol-contrib_0.139.0_linux_${ARCH}.tar.gz"
  tar -xvf "otelcol-contrib_0.139.0_linux_${ARCH}.tar.gz" otelcol-contrib
  sudo install -m 755 otelcol-contrib /usr/local/bin/otelcol-contrib
  ```

- The `otelcol-contrib` system user foundry runs the service as:

  ```bash
  sudo useradd -r -s /sbin/nologin otelcol-contrib
  ```

  Foundry proves both of these exist before casting; it never creates them for you.
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
    flavor: binary
    mode: systemd
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "http://<signoz-host>:4318"
```

The generated collector config follows the [VM Collection Agent configuration guide](https://signoz.io/docs/opentelemetry-collection-agents/vm/configure/) and exports to `${env:SIGNOZ_INGESTION_ENDPOINT}`.

### Point the agent at your SigNoz

`spec.collector.spec.env` becomes `Environment=` lines in the generated systemd unit:

```yaml
spec:
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "http://<signoz-host>:4318"
        OTEL_RESOURCE_ATTRIBUTES: "deployment.environment=production"
```

- For a Self-Hosted Community installation the OTLP HTTP ingest is port `4318` on the SigNoz host. Community has no ingestion key, so the endpoint is all the agent needs (see [Cloud to Self-Hosted](https://signoz.io/docs/ingestion/cloud-vs-self-hosted/#cloud-to-self-hosted)).
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
    flavor: binary
    mode: systemd
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

### Custom binary path

If the binary lives somewhere other than `/usr/local/bin/otelcol-contrib`, set the annotation:

```yaml
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/collector-binary-path: /opt/otelcol/otelcol-contrib
```

## Deploy

Point the endpoint in the casting at your SigNoz, then, as root (the unit writes to `/etc/otelcol-contrib` and drives `systemctl`):

```bash
sudo foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Validate prerequisites
sudo foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Deploy: writes /etc/otelcol-contrib/config.yaml, enables and starts the unit
sudo foundryctl cast -f casting.yaml
```

## Generated output

```text
pours/collectionagent/
  signoz-collector-agent.service
  collector/
    agent/
      agent.yaml
```

## After deployment

```bash
# Check the agent is healthy
curl -fsS localhost:13133/healthz && echo " OK"

# View agent logs
sudo journalctl -u signoz-collector-agent -f

# Stop the agent
sudo systemctl stop signoz-collector-agent
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) on the host at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

In SigNoz, the host appears under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/).

> [!IMPORTANT]
> **Upgrading:** re-running `cast` rewrites `/etc/otelcol-contrib/config.yaml` and the unit, then restarts the service so the new config takes effect immediately — no separate recreate step.

## Melt

```bash
sudo foundryctl melt -f casting.yaml
```

Stops and disables the unit; the poured unit file stays on disk, but `/etc/otelcol-contrib/config.yaml` (a definition foundry wrote, not accumulated data) is removed. The binary you placed and the `otelcol-contrib` user are untouched.

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones. For changes to the generated unit file itself, use [patches](../../../../concepts/patches.md).
