# Ledger

Foundryctl maintains an anonymous usage ledger to help the SigNoz team understand how the tool is used, identify common errors, and prioritize improvements. **No personally identifiable information (PII) is collected.**

## What is collected

Each command execution sends a single event with the following properties:

| Property | Description | Example |
|---|---|---|
| `kind` | Casting kind | `Installation`, `CollectionAgent` |
| `platform` | Deployment platform from casting.yaml | `aws`, `docker`, `linux` |
| `mode` | Deployment mode | `docker`, `systemd`, `kubernetes` |
| `flavor` | Deployment flavor | `compose`, `binary`, `helm` |
| `patches_count` | Number of patch entries | `0`, `2` |
| `infrastructure_enabled` | Whether IaC generation is enabled | `true` / `false` |
| `metastore_kind` | MetaStore backend type | `postgres`, `sqlite` |
| `telemetrystore_kind` | TelemetryStore backend type | `clickhouse` |
| `telemetrykeeper_kind` | TelemetryKeeper backend type | `clickhousekeeper` |
| `collector_kind` | Collector type (CollectionAgent castings only) | `agent` |
| `mcp_enabled` | Whether the MCP server molding is enabled | `true` / `false` |
| `success` | Whether the command succeeded | `true` / `false` |
| `error` | Error message (on failure only) | `missing tool: docker` |
| `error_type` | Error type (on failure only) | `invalid-input` |
| `error_cause` | Underlying error message (on failure only) | `missing tool: docker` |
| `os` | Operating system | `linux`, `darwin` |
| `arch` | CPU architecture | `amd64`, `arm64` |
| `foundry_version` | foundryctl version | `0.1.0` |
| `invoked_by` | Who invoked the command, detected from environment variables. `unknown` means undetected, not necessarily human | `agent`, `unknown` |
| `agent_name` | Detected AI agent name (only sent when `invoked_by` is `agent`) | `claude`, `cursor`, `codex` |

### Identity

Events are attributed using a hashed machine ID (HMAC-SHA256 of the OS machine ID with an application-specific salt). The hash is not reversible and cannot be correlated across different applications. No usernames, emails, IP addresses, hostnames, or file contents are sent.

## Tracked commands

Each command sends an event named `foundryctl: <command> <outcome>`, for example `foundryctl: forge succeeded` or `foundryctl: cast failed`:

- `gauge`
- `forge`
- `cast`
- `catalog`

## How to disable the ledger

### Per-command

Use the `--no-ledger` flag on any command:

```bash
foundryctl forge --no-ledger
foundryctl --no-ledger cast
```
