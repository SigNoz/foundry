# Docker Swarm

| Field | Value |
| --- | --- |
| **Mode** | `docker` |
| **Flavor** | `swarm` |
| **Platform** | `-` |

## Overview

Deploys SigNoz on a Docker Swarm cluster. Foundry generates a Compose file and deploys it as a stack using `docker stack deploy`.

## Prerequisites

- Docker Engine 20.10+ with Swarm mode initialized (`docker swarm init`)
- At least one manager node

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: swarm
    mode: docker
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Generate the compose file
foundryctl forge -f casting.yaml

# Deploy manually
docker stack deploy -c pours/deployment/compose.yaml signoz
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
# List services in the stack
docker stack services signoz

# View logs for a service
docker service logs signoz_signoz -f

# Remove the stack
docker stack rm signoz
```

## Customization

For platform-level changes to the generated `compose.yaml`, use [patches](../../../concepts/patches.md).
