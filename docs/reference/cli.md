# CLI Reference

`foundryctl` is the Foundry command-line tool for generating and deploying SigNoz.

## Usage

```
foundryctl [command] [flags]
```

## Global flags

| Flag | Description | Default |
|---|---|---|
| `-d`, `--debug` | Enable debug mode with verbose logging | `false` |
| `-f`, `--file` | Path to the casting configuration file | `casting.yaml` |
| `-p`, `--pours` | Directory for generated output | `./pours` |
| `-h`, `--help` | Help for foundryctl | |

> [!TIP]
> Use `--debug` when troubleshooting errors. It shows internal details that are hidden by default.

## Commands

### gauge

Validate that all required tools are installed for your deployment mode.

```bash
foundryctl gauge -f casting.yaml
```

Exits with an error if any required tool is missing. Run this before `forge` or `cast` to catch missing dependencies early.

### forge

Generate deployment and configuration files from your casting.

```bash
foundryctl forge -f casting.yaml -p ./pours
```

Reads the casting file, merges your overrides with defaults, and writes the generated files to the pours directory. The output depends on the deployment mode: Compose files for Docker, service units for systemd, Kubernetes manifests for Kustomize, Helm values for Helm, and so on.

After forging, a `casting.yaml.lock` file is written with checksums to track the current deployment state.

### cast

Deploy SigNoz to your target environment. Runs `gauge` and `forge` automatically before deploying.

```bash
foundryctl cast -f casting.yaml
```

Skip individual steps:

```bash
# Skip tool validation
foundryctl cast --no-gauge

# Skip file generation (use existing pours)
foundryctl cast --no-forge
```

> [!NOTE]
> Some deployment modes (Render, Coolify, Railway) do not support automated deployment. For these, `cast` generates the files and prints instructions for manual deployment on the target platform.

### gen

Generate example casting files and pours for all supported deployment modes.

```bash
# Generate example castings and forged output
foundryctl gen examples

# Generate JSON schemas
foundryctl gen schemas
```

This is the fastest way to get a working `casting.yaml` for your environment. Each example is written to `docs/examples/<deployment>/` with a minimal casting file and pre-forged output.
