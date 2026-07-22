# MCP server

The [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) lets AI clients (Claude Code, Claude Desktop, Cursor, and others) query your SigNoz telemetry over the [Model Context Protocol](https://modelcontextprotocol.io): logs, traces, metrics, dashboards, and alerts.

Foundry deploys it as a molding alongside the rest of the stack. It is optional and disabled by default; opt in from the casting.

## Enable it

Add an `mcp` block under `spec` and turn it on:

```yaml
spec:
  mcp:
    spec:
      enabled: true
```

The molding sets the rest automatically:

| Variable | Value | Why |
|---|---|---|
| `TRANSPORT_MODE` | `http` | Runs as a long-lived HTTP service, not the per-client stdio transport |
| `MCP_SERVER_PORT` | `8000` | Port the server listens on |
| `SIGNOZ_URL` | the co-located SigNoz apiserver | So the server knows which instance to query |

You can override these or set others (including OAuth) via `mcp.spec.env`, the same as any other molding.

## Authentication

In HTTP mode the SigNoz API key and URL only need to live in one place. Foundry sets `SIGNOZ_URL` on the server and leaves the key to the client, so the deployment never holds a secret. The server supports three modes:

| Mode | Server env | Client provides |
|---|---|---|
| Per-request header (default here) | `SIGNOZ_URL` | a `SIGNOZ-API-KEY` header |
| Credentials on server | `SIGNOZ_URL` + `SIGNOZ_API_KEY` | just the URL |
| OAuth (multi-tenant) | `OAUTH_ENABLED`, `OAUTH_TOKEN_SECRET`, `OAUTH_ISSUER_URL` | browser sign-in |

For a single self-hosted SigNoz, the per-request header is simplest: nothing secret in `casting.yaml`, and each user authenticates from their own client.

### Create a SigNoz API key

1. Open the SigNoz UI (for compose, `http://localhost:8080`).
2. Go to Settings -> API Keys (Admin only) and create a key.
3. Give it to your MCP client (below), not the casting.

## Connect a client

The MCP endpoint is the server URL plus `/mcp`. Where the server is reachable depends on the deployment mode:

| Mode | Endpoint |
|---|---|
| `docker`/`compose`, `docker`/`swarm` | `http://localhost:8000/mcp` (port `8000` is published) |
| `systemd`/`binary` | `http://<host>:8000/mcp` |
| `kubernetes`/`kustomize` | the `signoz-mcp` ClusterIP service on port `8000`; expose it outside the cluster per your setup |
| `render`/`blueprint` | the `signoz-mcp` web service URL from the Render dashboard, plus `/mcp` |
| `coolify`/`stack`, `railway`/`template` | expose port `8000` on the platform, then use that URL plus `/mcp` |

### Claude Code

Claude Code connects to HTTP MCP servers natively:

```bash
claude mcp add --transport http signoz-local http://localhost:8000/mcp \
  --header "SIGNOZ-API-KEY: <your-key>"
```

Verify with `/mcp` (or `claude mcp list`). Add `--scope user` (before the name) to use it across all your projects.

### Claude Desktop

Claude Desktop has no native remote transport, so bridge it with [`mcp-remote`](https://www.npmjs.com/package/mcp-remote) (requires Node and `npx`). Add a new entry to `claude_desktop_config.json` beside any existing servers, then fully quit and reopen Claude Desktop (Cmd+Q, not just the window):

```json
{
  "mcpServers": {
    "signoz-local": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://localhost:8000/mcp", "--header", "SIGNOZ-API-KEY:${KEY}"],
      "env": { "KEY": "<your-key>" }
    }
  }
}
```

## Next steps

- [Docker Compose with MCP example](../examples/docker/compose-mcp/README.md) - deploy SigNoz with MCP enabled, end to end
- [Moldings](moldings.md) - the components that make up a deployment
- [Casting](casting.md) - the full casting file structure
