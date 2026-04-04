# Docker Compose

| Field | Value |
| --- | --- |
| **Mode** | `docker` |
| **Flavor** | `compose` |
| **Platform** | `-` |

## Overview

Deploys SigNoz using Docker Compose with all Foundry defaults. This is the simplest way to run SigNoz locally or on a single node.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: compose
    mode: docker
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Validate prerequisites
foundryctl gauge -f casting.yaml

# Generate compose files
foundryctl forge -f casting.yaml

# Start the stack
cd pours/deployment && docker compose up -d
```

## Generated output

```text
pours/deployment/
  compose.yaml
  configs/
    ingester/
      ingester.yaml
      opamp.yaml
    telemetrykeeper/
      clickhousekeeper/
        keeper-0.yaml
    telemetrystore/
      clickhouse/
        config.yaml
        functions.yaml
```

## After deployment

```bash
# Check running containers
docker ps

# View logs for a specific service
docker compose -f pours/deployment/compose.yaml logs -f signoz

# Stop the stack
cd pours/deployment && docker compose down
```

> [!NOTE]
> - `foundryctl cast` detects whether `docker compose` (v2 plugin) or `docker-compose` (legacy standalone) is available and uses whichever it finds, preferring the v2 plugin.
> - The `metadata.name` is used as the Compose project name (`-p` flag).

## Customization

Override component images, replicas, or environment variables in the casting spec. For platform-level changes to the generated `compose.yaml` (memory limits, networks, volumes), use [patches](../../../concepts/patches.md).
