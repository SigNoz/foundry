# Dokploy Stack

| Field | Value |
| --- | --- |
| **Mode** | `-` |
| **Flavor** | `stack` |
| **Platform** | `dokploy` |

## Overview

Generates a Docker Swarm `compose.yaml` and component configuration files for deployment through Dokploy's stack feature.

> [!NOTE]
> `foundryctl cast` does not deploy to Dokploy automatically. It generates the files and prints instructions for manual deployment.

## Prerequisites

- A [Dokploy](https://dokploy.com) instance
- The external `dokploy-network` created by Dokploy

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: stack
    platform: dokploy
```

## Deploy

```bash
foundryctl forge -f casting.yaml
```

Add a domain for the generated SigNoz service in Dokploy, using container port `8080`, then deploy `pours/deployment/compose.yaml` with the stack feature.

## Generated output

```text
pours/deployment/
  compose.yaml
  ingester/
    ingester.yaml
    opamp.yaml
  telemetrykeeper/
    clickhousekeeper/
      keeper-0.yaml
  telemetrystore/
    clickhouse/
      config-0-0.yaml
      functions.yaml
```

## Customization

For changes to the generated `compose.yaml`, use [patches](../../../concepts/patches.md).
