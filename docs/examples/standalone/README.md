# Standalone Docker Image

| | |
|---|---|
| Casting | `systemd` / `binary` |
| Use Case | Single Docker image for quick testing, development, and CI |

## Overview

Runs SigNoz in a single Docker container using systemd casting. On first boot, `foundryctl cast` generates configs and starts all services.

## Quick Start

```bash
docker build -t signoz/signoz:standalone -f docs/examples/standalone/Dockerfile .
docker run -d --name signoz --privileged \
    -p 8080:8080 \
    -p 4317:4317 \
    -p 4318:4318 \
    signoz/signoz:standalone
```

Access SigNoz UI at `http://localhost:8080`.

Send telemetry to:
- OTLP gRPC: `localhost:4317`
- OTLP HTTP: `localhost:4318`

## Version Pinning

By default, the image pulls the latest release of each component. To pin specific versions:

```bash
docker build \
    --build-arg SIGNOZ_VERSION=v0.117.0 \
    --build-arg INGESTER_VERSION=v0.144.2 \
    --build-arg FOUNDRY_VERSION=v0.0.7 \
    -t signoz/signoz:standalone \
    -f docs/examples/standalone/Dockerfile .
```


## How It Works

1. Container starts with systemd as PID 1
2. `foundry-setup.service` runs `foundryctl cast` on first boot
3. Cast pipeline: forge (generate service units + configs) → deploy (enable + start services)

## Mount Volumes

```bash
docker run -d --name signoz --privileged \
    -p 8080:8080 \
    -p 4317:4317 \
    -p 4318:4318 \
    -v signoz-clickhouse:/var/lib/clickhouse \
    -v signoz-data:/var/lib/signoz \
    signoz/signoz:standalone
```

## Logs

```bash
# All services
docker exec signoz journalctl -f

# Specific service
docker exec signoz journalctl -u signoz-signoz.service -f
docker exec signoz journalctl -u signoz-ingester.service -f
docker exec signoz journalctl -u signoz-telemetrystore-clickhouse-0-0.service -f
```

## Limitations

- Requires `--privileged` flag (systemd needs cgroup access)
- `docker logs` is empty — use `journalctl` inside the container
- Single-node only (no clustering)
