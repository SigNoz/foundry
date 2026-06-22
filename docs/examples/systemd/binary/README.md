# Systemd Binary Casting

| Field | Value |
| --- | --- |
| **Mode** | `systemd` |
| **Flavor** | `binary` |
| **Platform** | `-` |

## Overview

Deploys SigNoz on bare metal as systemd units — one service per component (SigNoz, OTel Collector, ClickHouse, ClickHouse Keeper, and PostgreSQL when used), running under a dedicated `signoz` service user. `foundryctl cast` generates the units and starts them.

## Prerequisites

Install the component binaries below; `cast` handles everything else. They are expected at standard locations — install elsewhere and point to them with [annotations](#annotations).

- [SigNoz](https://github.com/SigNoz/signoz/releases/latest)
- [SigNoz OTel Collector](https://github.com/SigNoz/signoz-otel-collector/releases/latest)
- [ClickHouse](https://clickhouse.com/docs/install) — a single `clickhouse` binary serves both the telemetry store and the keeper
- [PostgreSQL](https://www.postgresql.org/download/) — only when `metastore.kind` is `postgres`

> [!IMPORTANT]
> Install SigNoz by extracting the **full** release tarball into `/opt/signoz` — do not move the `signoz` binary on its own. It resolves the web UI and email/alert templates relative to itself, so `bin/`, `web/`, `templates/`, and `conf/` must stay together:
>
> ```bash
> ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
> sudo mkdir -p /opt/signoz
> curl -fsSL "https://github.com/SigNoz/signoz/releases/latest/download/signoz_linux_${ARCH}.tar.gz" \
>   | sudo tar -xz --strip-components=1 -C /opt/signoz
> ```

## Configuration

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: binary
    mode: systemd
```

## Deploy

```bash
sudo foundryctl cast -f casting.yaml
```

`cast` checks the binaries are present, then installs and starts the systemd units.

> [!NOTE]
> `cast` requires `sudo` because it manages systemd services, creates the service users, and writes to system directories.

To inspect before deploying:

```bash
# Check the orchestration tool (systemctl) is available
foundryctl gauge -f casting.yaml

# Generate the unit files and configs into pours/ without touching the host
foundryctl forge -f casting.yaml
```

## Generated output

```text
pours/deployment/
  signoz-ingester.service
  signoz-metastore-postgres.service
  signoz-signoz.service
  signoz-telemetrykeeper-clickhousekeeper-0.service
  signoz-telemetrystore-clickhouse-0-0.service
  signoz-telemetrystore-migrator.service
  ingester/
    ingester.yaml
    opamp.yaml
  telemetrykeeper/clickhousekeeper/
    keeper-0.yaml
  telemetrystore/clickhouse/
    config-0-0.yaml
    functions.yaml
```

## Operating

Services are named `<metadata.name>-<component>.service`.

```bash
# Status of all services
systemctl status 'signoz-*'

# Follow logs for one service
journalctl -u signoz-signoz.service -f

# Follow logs for all services
journalctl -u 'signoz-*' -f
```

## Annotations

Set an annotation only when a binary is installed outside its default location.

| Annotation | Default | Binary |
| --- | --- | --- |
| `foundry.signoz.io/signoz-binary-path` | `/opt/signoz/bin/signoz` | SigNoz |
| `foundry.signoz.io/ingester-binary-path` | `/opt/ingester/bin/signoz-otel-collector` | OTel Collector |
| `foundry.signoz.io/metastore-postgres-binary-path` | `/usr/bin/postgres` | PostgreSQL (its directory must also hold `initdb`) |
| `foundry.signoz.io/telemetrystore-clickhouse-binary-path` | `/usr/bin/clickhouse` | ClickHouse, run as `clickhouse server` |
| `foundry.signoz.io/telemetrykeeper-clickhousekeeper-binary-path` | `/usr/bin/clickhouse` | ClickHouse, run as `clickhouse keeper` |

```yaml
metadata:
  annotations:
    foundry.signoz.io/signoz-binary-path: /custom/bin/signoz
```
