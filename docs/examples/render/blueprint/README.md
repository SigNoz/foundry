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

## MCP server (optional)

Foundry can deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) alongside the stack so AI clients can query your telemetry. It is disabled by default; enable it in the casting:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: blueprint
    platform: render
  mcp:
    spec:
      enabled: true
```

This adds a `signoz-mcp` web service to the blueprint with its own public URL; the MCP endpoint is that URL plus `/mcp`.

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

For changes to the generated `render.yaml`, use [patches](../../../concepts/patches.md).
