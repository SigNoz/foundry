# Railway Template

| Field | Value |
| --- | --- |
| **Mode** | `-` |
| **Flavor** | `template` |
| **Platform** | `railway` |

## Overview

Generates per-component Dockerfiles, `railway.json` service definitions, and config files for deploying SigNoz on Railway. Deployment is manual via the Railway dashboard.

> [!NOTE]
> `foundryctl cast` does not deploy to Railway automatically. It generates the files and prints instructions for manual deployment.

## Prerequisites

- A [Railway](https://railway.app) account

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: template
    platform: railway
```

## Deploy

```bash
# Generate the template files
foundryctl forge -f casting.yaml
```

After forging, use the generated files in `pours/deployment/` to create services on Railway.

## Generated output

```text
pours/deployment/
  telemetrykeeper/
    Dockerfile
    railway.json
    keeper.d/
  telemetrystore/
    Dockerfile
    railway.json
    config.d/
  metastore/
    Dockerfile
    railway.json
  signoz/
    Dockerfile
    railway.json
  ingester/
    Dockerfile
    railway.json
  migrator/
    Dockerfile
    railway.json
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
    flavor: template
    platform: railway
  mcp:
    spec:
      enabled: true
```

This adds an `mcp/` service (Dockerfile and `railway.json`) to the generated template. Configure public networking for the service on port `8000` in Railway to reach it from AI clients; the MCP endpoint is that URL plus `/mcp`.

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

For changes to the generated files, use [patches](../../../concepts/patches.md).
