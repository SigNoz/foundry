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

### 2. Install PostgreSQL

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

### 4. Install SigNoz OTel Collector (Ingester)

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


### Custom Binary Paths

If foundryctl is unable to identify the binary paths for postgres, signoz, or ingester, specify them using the `*_BINARY_PATH` environment variables:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: binary
    mode: systemd
  signoz:
    spec:
      env:
        SIGNOZ_BINARY_PATH: /opt/signoz/bin/signoz
  ingester:
    spec:
      env:
        OTEL_COLLECTOR_BINARY_PATH: /opt/ingester/bin/signoz-otel-collector
  metastore:
    spec:
      env:
        POSTGRES_BINARY_PATH: /usr/bin/postgres
```