# Render Blueprint

| Field | Value |
| --- | --- |
| **Mode** | `-` |
| **Flavor** | `blueprint` |
| **Platform** | `render` |

## Overview

Generates a Render Blueprint (`render.yaml`) and supporting Dockerfiles for deploying SigNoz on the Render cloud platform. Deployment is manual via Render's Infrastructure as Code flow.

> [!NOTE]
> `foundryctl cast` does not deploy to Render automatically. It generates the files and prints instructions for manual deployment.

## Prerequisites

- A [Render](https://render.com) account

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: blueprint
    platform: render
```

## Deploy

```bash
# Generate the blueprint and supporting files
foundryctl forge -f casting.yaml
```

After forging, deploy the generated `render.yaml` to Render using [Infrastructure as Code](https://render.com/docs/infrastructure-as-code#setup).

## Generated output

```text
pours/deployment/
  render.yaml
  configs/
    telemetrykeeper/
      Dockerfile
      keeper.d/
    telemetrystore/
      Dockerfile
      config.d/
    ingester/
      Dockerfile
```

## Customization

For changes to the generated `render.yaml`, use [patches](../../../concepts/patches.md).
