# Systemd Binary Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `systemd` |
| **Flavor** | `binary` |

## Overview

Deploys a SigNoz [VM Collection Agent](https://signoz.io/docs/opentelemetry-collection-agents/vm/overview/) on a Linux host as a single systemd service running the OpenTelemetry Collector Contrib binary you place yourself. The agent collects the host's telemetry and exports it, along with anything your applications send it, to any SigNoz: Self-Hosted Community, Self-Hosted Enterprise, or SigNoz Cloud. It runs directly on the host, so the `host.name` resource attribute on all telemetry is the real host's name.

It collects:

- Host metrics: CPU, memory, disk and filesystem, network, load, paging, process counts and per-process usage
- Host log files under `/var/log`: system, auth, kernel, cron and application `.log` files, read from the end of each file
- Host and cloud identity: hostname, operating system, and cloud metadata on EC2, GCP, Azure and ECS when present
- OTLP intake for your applications on `localhost:4317` (gRPC) and `localhost:4318` (HTTP)

## Prerequisites

- A Linux host running systemd 232 or newer
- Root on that host: `cast` installs a unit, creates the service user and drives `systemctl`
- The [OpenTelemetry Collector Contrib binary](https://github.com/open-telemetry/opentelemetry-collector-releases/releases) placed at `/opt/signoz-collector-agent/bin/otelcol-contrib`, or wherever the [annotation](#annotations) points:

  ```bash
  ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
  curl -fsSL "https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.139.0/otelcol-contrib_0.139.0_linux_${ARCH}.tar.gz" | tar -xz otelcol-contrib
  sudo install -D -m 755 otelcol-contrib /opt/signoz-collector-agent/bin/otelcol-contrib
  ```

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

Set the endpoint and environment through `spec.collector.spec.env`, which becomes the environment file the unit reads:

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

The casting is the single input; treat it as sensitive once the key is in it. The environment file is installed readable by root only.

### Custom binary path

When the binary lives somewhere other than `/opt/signoz-collector-agent/bin/otelcol-contrib`, point at it with an annotation:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/collector-binary-path: /opt/otelcol/otelcol-contrib
spec:
  deployment:
    flavor: binary
    mode: systemd
```

## Deploy

Point the endpoint in the casting at your SigNoz, then:

```bash
# Validate prerequisites
sudo foundryctl gauge -f casting.yaml

# Generate the deployment files
foundryctl forge -f casting.yaml

# Install the files and start the agent
sudo foundryctl cast -f casting.yaml
```

## Generated output

```text
pours/collectionagent/
  collector/
    agent/
      agent.yaml
      signoz-collector-agent.conf
      signoz-collector-agent.service
      signoz-collector-agent.sysusers.conf
```

`agent.yaml` is the collector config, `signoz-collector-agent.conf` the environment file, `signoz-collector-agent.service` the unit, and `signoz-collector-agent.sysusers.conf` the service-account declaration. `cast` copies them to `/etc/signoz-collector-agent`, `/etc/systemd/system` and `/etc/sysusers.d`, creates the service user, and enables and restarts the service.

## Operating

```bash
# Check the agent is healthy
curl -fsS localhost:13133/healthz && echo " OK"

# Check the service is running
sudo systemctl status signoz-collector-agent

# View agent logs
sudo journalctl -u signoz-collector-agent -f

# Stop the agent
sudo systemctl stop signoz-collector-agent
```

Point [instrumented applications](https://signoz.io/docs/instrumentation/) on the host at the agent with `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317` (gRPC) or `http://localhost:4318` (HTTP). The host then appears in SigNoz under Infrastructure Monitoring. Re-running `cast` reinstalls the files and restarts the service, so a changed casting takes effect immediately.

**Removal.** Disable the service, remove the installed files, and reload systemd:

```bash
sudo systemctl disable --now signoz-collector-agent
sudo rm /etc/systemd/system/signoz-collector-agent.service /etc/sysusers.d/signoz-collector-agent.conf
sudo rm -r /etc/signoz-collector-agent
sudo systemctl daemon-reload
```

The `signoz-collector-agent` service user stays behind; remove it with `userdel` when nothing else needs it.

## Customization

Two knobs, one for each half of the deployment. `spec.collector.spec.config.data` merges YAML into the generated collector config, keyed by the poured file's path; user keys win over generated ones, and a list replaces the generated list wholesale rather than appending to it. `spec.patches` applies [JSON Patch](../../../../concepts/patches.md) to a poured file; the unit is poured as INI, whose JSON shape is section names at the top level and directives inside them, so the path to a `[Service]` directive is `/Service/<Directive>`.

### Add application log files

An application that writes its log files outside `/var/log` and `/var/adm` is added by re-stating the `include` list with the extra path on it:

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
      config:
        data:
          collector/agent/agent.yaml: |
            receivers:
              filelog/varlog:
                include:
                  - /var/log/*.log
                  - /var/log/*log
                  - /var/log/*messages*
                  - /var/log/*secure*
                  - /var/log/*auth*
                  - /var/log/*mesg
                  - /var/log/*cron
                  - /var/log/*acpid
                  - /var/log/*.out
                  - /var/adm/*.log
                  - /var/adm/*messages*
                  - /var/log/nginx/*.log
```

The eleven shipped paths are re-listed alongside the new one: leave one out and it stops being tailed. The poured `agent.yaml` then carries exactly that `include` list, with the `exclude` list untouched.

### Read the systemd journal

On a host without `rsyslog`, Debian 12+ and Ubuntu 24.04 among them, the journal is the system log and nothing under `/var/log` holds it. Turning it on takes an added source in the logs pipeline and the `systemd-journal` group on the unit, so the agent may read it:

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
      config:
        data:
          collector/agent/agent.yaml: |
            receivers:
              journald:
                priority: info
                start_at: end
            service:
              pipelines:
                logs:
                  receivers: [otlp/http, otlp/grpc, filelog/varlog, journald]
  patches:
    - target: "collectionagent/collector/agent/signoz-collector-agent.service"
      operations:
        - op: add
          path: /Service/SupplementaryGroups
          value: systemd-journal
```

The logs pipeline re-lists every source the agent already had, and the poured unit then carries, in `[Service]`:

```ini
SupplementaryGroups=systemd-journal
```

## Annotations

| Annotation | Default | Binary |
| --- | --- | --- |
| `foundry.signoz.io/collector-binary-path` | `/opt/signoz-collector-agent/bin/otelcol-contrib` | OpenTelemetry Collector Contrib |

Set it only when the binary is installed outside that default.
