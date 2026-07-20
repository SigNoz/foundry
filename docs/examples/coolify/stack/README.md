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

## MCP server (optional)

Foundry can deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) alongside the stack so AI clients can query your telemetry. It is disabled by default; enable it in the casting:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: stack
    platform: coolify
  mcp:
    spec:
      enabled: true
```

This adds a `signoz-mcp` service to the stack listening on port `8000`. Expose that port from the Coolify dashboard to reach it from AI clients; the MCP endpoint is the exposed URL plus `/mcp`.

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

For changes to the generated `coolify.yaml`, use [patches](../../../concepts/patches.md).
