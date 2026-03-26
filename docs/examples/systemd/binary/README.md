# Systemd Binary Casting

| Field | Value |
| --- | --- |
| **Mode** | `systemd` |
| **Flavor** | `binary` |
| **Platform** | `-` |

## Overview

Deploys SigNoz on bare metal using systemd service units. Each SigNoz component runs as a separate systemd service under a dedicated `signoz` user. Foundry generates the service files and config directories, and `foundryctl cast` installs and starts them.

## Prerequisites

- [ClickHouse](https://clickhouse.com/docs/en/install) (`clickhouse-server` and `clickhouse-keeper`)
- [PostgreSQL](https://www.postgresql.org/download/) for the metadata store
- [SigNoz binary](https://github.com/SigNoz/signoz/releases/latest) installed to `/opt/signoz`
- [SigNoz OTel Collector binary](https://github.com/SigNoz/signoz-otel-collector/releases/latest) installed to `/opt/ingester`
- A `signoz` system user with ownership of binary and data directories:

```bash
sudo useradd -r -s /sbin/nologin signoz
sudo chown -R signoz:signoz /opt/signoz /var/lib/signoz /opt/ingester /var/lib/ingester
```

The `signoz` user must also have traverse permissions to the `pours/` output directory.

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: binary
    mode: systemd
```

## Deploy

Run the full pipeline (validate prerequisites, generate files, install and start services):

```bash
sudo foundryctl cast -f casting.yaml
```

Step-by-step alternative:

```bash
# 1. Validate prerequisites
foundryctl gauge -f casting.yaml

# 2. Generate deployment files
foundryctl forge -f casting.yaml

# 3. Install and start services manually from pours/deployment/
sudo cp pours/deployment/*.service /etc/systemd/system/
sudo cp -r pours/deployment/configs/ /etc/signoz/
sudo systemctl daemon-reload
sudo systemctl enable --now signoz-*.service
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
  configs/
    ingester/
      ingester.yaml
      opamp.yaml
    telemetrykeeper/
      keeper-0.yaml
    telemetrystore/
      config.yaml
      functions.yaml
```

## After deployment

Check service status (replace `signoz` with your `metadata.name`):

```bash
systemctl status signoz-signoz.service
systemctl status signoz-ingester.service
systemctl status signoz-telemetrystore-clickhouse-0-0.service
systemctl status signoz-telemetrykeeper-clickhousekeeper-0.service
systemctl status signoz-metastore-postgres.service
```

View logs for a specific service:

```bash
journalctl -u signoz-signoz.service -f
```

View logs for all SigNoz services:

```bash
journalctl -u 'signoz-*' -f
```

## Annotations

Use annotations to specify custom binary paths when binaries are not installed in the default locations.

| Annotation | Type | Description |
| --- | --- | --- |
| `foundry.signoz.io/signoz-binary-path` | `string` | Path to the SigNoz binary |
| `foundry.signoz.io/ingester-binary-path` | `string` | Path to the OTel Collector binary |
| `foundry.signoz.io/metastore-postgres-binary-path` | `string` | Path to the PostgreSQL binary |

Example with custom binary paths:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/signoz-binary-path: /opt/signoz/bin/signoz
    foundry.signoz.io/ingester-binary-path: /opt/ingester/bin/signoz-otel-collector
    foundry.signoz.io/metastore-postgres-binary-path: /usr/bin/postgres
spec:
  deployment:
    flavor: binary
    mode: systemd
```
