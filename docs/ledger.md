# Ledger

Foundryctl maintains an anonymous usage ledger to help the SigNoz team understand how the tool is used, identify common errors, and prioritize improvements. **No personally identifiable information (PII) is collected.**

## What is collected

Each command execution sends a single event. The event carries the properties of every casting document in the casting file, then the outcome of the command.

### Casting documents

| Property | Description | Example |
|---|---|---|
| `kinds` | Kinds of the casting documents in the file | `["Installation", "CollectionAgent"]` |
| `platform` | Installation deployment platform | `aws`, `docker`, `linux` |
| `mode` | Installation deployment mode | `docker`, `systemd`, `kubernetes` |
| `flavor` | Installation deployment flavor | `compose`, `binary`, `helm` |
| `patches_count` | Number of Installation patch entries | `0`, `2` |
| `infrastructure_bound` | Whether the Installation is bound to an Infrastructure casting | `true` / `false` |
| `metastore_kind` | MetaStore backend type | `postgres`, `sqlite` |
| `telemetrystore_kind` | TelemetryStore backend type | `clickhouse` |
| `telemetrykeeper_kind` | TelemetryKeeper backend type | `clickhousekeeper` |
| `mcp_enabled` | Whether the MCP server molding is enabled | `true` / `false` |
| `collectionagent_<collector>_platform` | Deployment platform of the CollectionAgent with that collector kind (`agent`, `deployment` or `sidecar`) | `aws` |
| `collectionagent_<collector>_mode` | Deployment mode of that CollectionAgent | `docker`, `kubernetes`, `ecs` |
| `collectionagent_<collector>_flavor` | Deployment flavor of that CollectionAgent | `compose`, `kustomize`, `terraform` |
| `collectionagent_<collector>_patches_count` | Number of patch entries of that CollectionAgent | `0`, `1` |
| `infrastructure_platform` | Infrastructure deployment platform | `ecs` |
| `infrastructure_mode` | Infrastructure deployment mode | `ec2` |
| `infrastructure_flavor` | Infrastructure deployment flavor | `terraform` |
| `infrastructure_patches_count` | Number of Infrastructure patch entries | `0`, `1` |

Each casting document sends its own properties: the Installation's unprefixed, each CollectionAgent's under its collector kind, and the Infrastructure's under `infrastructure_`.

### Outcome

| Property | Description | Example |
|---|---|---|
| `success` | Whether the command succeeded | `true` / `false` |
| `error` | Error message (on failure only) | `missing tool: docker` |
| `error_type` | Error type (on failure only) | `invalid-input` |
| `error_cause` | Underlying error message (on failure only) | `missing tool: docker` |
| `failed_kind` | Kind of the casting document that failed the command (on failure only) | `CollectionAgent` |
| `failed_collector_kind` | Collector kind of the CollectionAgent that failed the command (on failure only) | `deployment` |

### Environment

| Property | Description | Example |
|---|---|---|
| `os` | Operating system | `linux`, `darwin` |
| `arch` | CPU architecture | `amd64`, `arm64` |
| `foundry_version` | foundryctl version | `0.1.0` |
| `invoked_by` | Who invoked the command, detected from environment variables. `unknown` means undetected, not necessarily human | `agent`, `unknown` |
| `agent_name` | Normalized AI agent name (only sent when `invoked_by` is `agent`) | `claude`, `cursor`, `codex` |
| `agent_fullname` | Full self-declared agent identifier (only sent when `invoked_by` is `agent`) | `claude-code_2-1-161_agent` |

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
