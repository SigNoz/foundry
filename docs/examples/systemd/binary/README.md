# Systemd Binary Casting

This guide explains how to use systemd binary casting for deploying SigNoz.

## Prerequisites

Before running `foundryctl cast`, install the required dependencies.

### 1. Install ClickHouse

ClickHouse is used as the telemetry store. Install both `clickhouse-server` and `clickhouse-keeper`.

- [ClickHouse Installation Guide](https://clickhouse.com/docs/en/install)

Verify installation:

```bash
clickhouse-server --version
clickhouse-keeper --version
```

### 2. Install Metastore Binary (PostgreSQL)

PostgreSQL is used as the metadata store.

- [PostgreSQL Installation Guide](https://www.postgresql.org/download/)

Verify installation:

```bash
postgres --version
```

### 3. Install SigNoz Binary

```bash
curl -L https://github.com/SigNoz/signoz/releases/latest/download/signoz_linux_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g').tar.gz -o signoz.tar.gz
tar -xzf signoz.tar.gz

sudo mkdir -p /opt/signoz /var/lib/signoz
sudo cp -r signoz_linux_*/* /opt/signoz/
```

### 4. Install Ingester Binary (SigNoz OTel Collector)

```bash
curl -L https://github.com/SigNoz/signoz-otel-collector/releases/latest/download/signoz-otel-collector_linux_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g').tar.gz -o signoz-otel-collector.tar.gz
tar -xzf signoz-otel-collector.tar.gz

sudo mkdir -p /opt/ingester /var/lib/ingester
sudo cp -r signoz-otel-collector_linux_*/* /opt/ingester/
```

### 5. Create signoz User

```bash
sudo useradd -r -s /sbin/nologin signoz
sudo chown -R signoz:signoz /opt/signoz /var/lib/signoz /opt/ingester /var/lib/ingester
```

Also, make sure that "signoz" user is allowed to transverse to the pours directory.

## Deployment

Create a `casting.yaml` file:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: binary
    mode: systemd
```

### 1. Verify Prerequisites

```bash
foundryctl gauge -f casting.yaml
```

### 2. Deploy SigNoz

```bash
sudo foundryctl cast -f casting.yaml
```

### 3. Verify Services

Replace `<name>` with your `metadata.name` from `casting.yaml`:

```bash
systemctl status <name>-signoz.service
systemctl status <name>-ingester.service
systemctl status <name>-telemetrystore-clickhouse-0-0.service
systemctl status <name>-telemetrykeeper-clickhousekeeper-0.service
systemctl status <name>-metastore-postgres.service
```

View logs:

```bash
journalctl -u <name>-signoz.service -f
```


## Configuration

### Custom Binary Path

Use annotations to specify custom binary paths or other deployment metadata:

| Name | Type | Description |
|------|------|-------------|
| `foundry.signoz.io/signoz-binary-path` | string | Path to the SigNoz binary |
| `foundry.signoz.io/ingester-binary-path` | string | Path to the OTel Collector binary |
| `foundry.signoz.io/metastore-postgres-binary-path` | string | Path to the PostgreSQL binary |

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
        foundry.signoz.io/signoz-binary-path: /opt/signoz/bin/signoz
        foundry.signoz.io/ingester-binary-path: /opt/ingester/bin/signoz-otel-collector
        foundry.signoz.io/metastore-binary-path: /usr/bin/postgres
spec:
  deployment:
    flavor: binary
    mode: systemd
```