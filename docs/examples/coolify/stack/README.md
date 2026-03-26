# Coolify Stack

| Field | Value |
| --- | --- |
| **Mode** | `-` |
| **Flavor** | `stack` |
| **Platform** | `coolify` |

## Overview

Generates a `coolify.yaml` stack definition for deploying SigNoz on a Coolify-managed server. Deployment is manual via the Coolify dashboard.

> [!NOTE]
> `foundryctl cast` does not deploy to Coolify automatically. It generates the files and prints instructions for manual deployment.

## Prerequisites

- A [Coolify](https://coolify.io) instance (self-hosted or cloud)

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: stack
    platform: coolify
```

## Deploy

```bash
# Generate the stack definition
foundryctl forge -f casting.yaml
```

After forging, deploy the generated `coolify.yaml` using the [Coolify stack feature](https://coolify.io/docs/knowledge-base/docker/compose).

## Generated output

```text
pours/deployment/
  coolify.yaml
```

## Customization

For changes to the generated `coolify.yaml`, use [patches](../../concepts/patches.md).
