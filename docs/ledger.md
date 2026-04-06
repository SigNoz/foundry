# Ledger

Foundryctl maintains an anonymous usage ledger to help the SigNoz team understand how the tool is used, identify common errors, and prioritize improvements. **No personally identifiable information (PII) is collected.**

## What is collected

Each command execution sends a single event with the following properties:

| Property | Description | Example |
|---|---|---|
| `platform` | Deployment platform from casting.yaml | `aws`, `docker`, `linux` |
| `mode` | Deployment mode | `docker`, `systemd`, `kubernetes` |
| `flavor` | Deployment flavor | `compose`, `binary`, `helm` |
| `patches_configured` | Whether patches are defined | `true` / `false` |
| `patch_count` | Number of patch entries | `0`, `2` |
| `infrastructure_enabled` | Whether IaC generation is enabled | `true` / `false` |
| `metastore_kind` | MetaStore backend type | `postgres`, `sqlite` |
| `telemetry_store_kind` | TelemetryStore backend type | `clickhouse` |
| `telemetry_keeper_kind` | TelemetryKeeper backend type | `clickhousekeeper` |
| `success` | Whether the command succeeded | `true` / `false` |
| `error` | Error message (on failure only) | `missing tool: docker` |
| `os` | Operating system | `linux`, `darwin` |
| `arch` | CPU architecture | `amd64`, `arm64` |
| `foundry_version` | foundryctl version | `0.1.0` |

### Identity

Events are attributed to the machine hostname as an anonymous identifier. No usernames, emails, IP addresses, or file contents are sent.

## Tracked commands

All commands send the same `foundryctl` event, differentiated by the `command` property:

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
