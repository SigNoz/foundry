# Systemd DEB Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `systemd` |
| **Flavor** | `deb` |

## Overview

Deploys a SigNoz [VM Collection Agent](https://signoz.io/docs/opentelemetry-collection-agents/vm/overview/) on a Debian or Ubuntu host, on top of the OpenTelemetry Collector Contrib `.deb` package. The package owns the `otelcol-contrib` binary and its systemd unit; foundry's managed surface is a systemd drop-in (`otelcol-contrib.service.d/foundry.conf`) plus the collector config at `/etc/otelcol-contrib/config.yaml`. The agent collects the host's telemetry and exports it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud.

- Host metrics (CPU, memory, disk, filesystem, load, network, paging, process, NFS) through the `hostmetrics` receiver, running natively on the host
- OTLP intake for your applications on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

The collector runs directly on the host, so the `host.name` resource attribute on all telemetry is the real host's name.

## Prerequisites

- A Debian or Ubuntu (or compatible) host with `systemd`
- The OpenTelemetry Collector Contrib `.deb` package installed:

  ```bash
  sudo apt-get update
  ARCH=$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')
  curl -sSL -O "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.139.0/otelcol-contrib_0.139.0_linux_${ARCH}.deb"
  sudo dpkg -i "otelcol-contrib_0.139.0_linux_${ARCH}.deb"
  ```

  Foundry proves the package's `otelcol-contrib.service` unit exists before casting; it never installs the package for you.
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
    flavor: deb
    mode: systemd
  collector:
    kind: agent
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: "http://<signoz-host>:4318"
```

The generated collector config follows the [VM Collection Agent configuration guide](https://signoz.io/docs/opentelemetry-collection-agents/vm/configure/) and exports to `${env:SIGNOZ_INGESTION_ENDPOINT}`.

### Point the agent at your SigNoz

`spec.collector.spec.env` becomes `Environment=` lines in the generated drop-in:

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
    flavor: deb
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

## Deploy

Point the endpoint in the casting at your SigNoz, then, as root (foundry writes to `/etc/otelcol-contrib` and `/etc/systemd/system/otelcol-contrib.service.d`, and drives `systemctl`):

```bash
sudo foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Validate prerequisites (the package's unit must already exist)
sudo foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Deploy: writes config.yaml + the drop-in, restarts otelcol-contrib.service
sudo foundryctl cast -f casting.yaml
```

## Generated output

```text
pours/collectionagent/
  otelcol-contrib.service.d/
    foundry.conf
  collector/
    agent/
      agent.yaml
```

## After deployment

```bash
# Check the agent is healthy
curl -fsS localhost:13133/healthz && echo " OK"

# View agent logs
sudo journalctl -u otelcol-contrib -f

# See the merged unit (package unit + foundry's drop-in)
systemctl cat otelcol-contrib.service
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) on the host at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP).

In SigNoz, the host appears under [Infrastructure Monitoring](https://signoz.io/docs/infrastructure-monitoring/hostmetrics/).

> [!IMPORTANT]
> **Upgrading:** re-running `cast` rewrites `config.yaml` and the drop-in, then restarts `otelcol-contrib.service` so the new config takes effect immediately.

## Melt

```bash
sudo foundryctl melt -f casting.yaml
```

Removes foundry's drop-in first, then stops and disables `otelcol-contrib.service` (its daemon-reload picks up the removal). `/etc/otelcol-contrib/config.yaml` stays: it is a package conffile the `.deb` shipped, not something foundry recorded, so foundry cannot restore it on melt. The package itself is never uninstalled; use `apt purge otelcol-contrib` to remove it.

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user keys win over generated ones. For changes to the generated drop-in itself, use [patches](../../../../concepts/patches.md).
