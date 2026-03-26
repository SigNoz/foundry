# Standalone Docker Image

| | |
| --- | --- |
| **Casting** | `systemd` / `binary` |
| **Use Case** | Single Docker image for quick testing, development, and CI |

## Overview

An OpenTelemetry-native observability backend in a single Docker image. The standalone image bundles all SigNoz components into one container for development, demo, and testing environments.

**Included components:**

- **SigNoz** - query engine and UI
- **OpenTelemetry Collector** - telemetry ingestion
- **ClickHouse** - telemetry storage
- **SQLite** - metadata storage
- **[Foundry](https://github.com/SigNoz/foundry)** - deployment orchestration via `foundryctl`

Applications can send telemetry using OpenTelemetry's standard defaults (OTLP gRPC/HTTP) without additional configuration. On first boot, Foundry generates all service configs and starts components via systemd.

## Prerequisites

- Docker Engine 20.10+

## Deploy

```bash
docker run -d --name signoz --privileged \
    -p 8080:8080 \
    -p 4317:4317 \
    -p 4318:4318 \
    signoz/signoz-standalone:latest
```

Access SigNoz UI at `http://localhost:8080`.

Send telemetry to:

- OTLP gRPC: `localhost:4317`
- OTLP HTTP: `localhost:4318`

## Customization

To customize the deployment, mount your own `casting.yaml` into the container:

```bash
docker run -d --name signoz --privileged \
    -p 8080:8080 \
    -p 4317:4317 \
    -p 4318:4318 \
    -v ./casting.yaml:/etc/foundry/casting.yaml \
    signoz/signoz-standalone:latest
```

See the default [casting.yaml](casting.yaml) for the full config structure.

## Persist Data

```bash
docker run -d --name signoz --privileged \
    -p 8080:8080 \
    -p 4317:4317 \
    -p 4318:4318 \
    -v signoz-clickhouse:/var/lib/clickhouse \
    -v signoz-data:/var/lib/signoz \
    signoz/signoz-standalone:latest
```

## After deployment

```bash
# View logs for all services
docker exec signoz journalctl -f

# View logs for a specific service
docker exec signoz journalctl -u signoz-signoz.service -f
docker exec signoz journalctl -u signoz-ingester.service -f
docker exec signoz journalctl -u signoz-telemetrystore-clickhouse-0-0.service -f
```

## Limitations

- Requires `--privileged` flag (systemd needs cgroup access)
- `docker logs` is empty - use `journalctl` inside the container
- Single-node only (no clustering)
