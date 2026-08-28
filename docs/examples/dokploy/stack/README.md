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

Set `SIGNOZ_DOMAIN` in Dokploy to the hostname that should serve the SigNoz UI, then deploy `pours/deployment/compose.yaml` with the stack feature. The generated Swarm labels configure HTTPS routing through Dokploy's Traefik service on container port `8080`. The SigNoz UI is the only service attached to Dokploy's shared network; databases remain isolated on the stack's private overlay network.

The ingester ports are exposed to container networks but are not published directly on the host. Connect collectors running in the Swarm to the stack's private overlay network. To receive telemetry from outside the Swarm, use a patch to attach the ingester to an appropriate ingress network and configure routing for port `4318` (OTLP/HTTP).

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
