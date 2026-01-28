# Foundry

Foundry is a centralized hub for [SigNoz](https://signoz.io) installation configurations and deployments - **integrations for install**. Select yours, configure, and run SigNoz.

## Overview

Just as a metalworking foundry turns raw materials into finished products, Foundry forges your deployment from a single configuration and casts SigNoz to fit your environment.

Foundry abstracts away the complexities of the installation process so you can spend time *using* SigNoz rather than *installing* it.

## Features

- **Multi-platform support**: Deploy SigNoz using Docker Compose, Systemd (bare metal), or Render for flexible installation across environments.
- **Single configuration file**: Configure your entire SigNoz stack with one concise file.
- **Automatic dependency management**: Handles inter-service dependencies
- **Tool validation**: Verify prerequisites before deployment

## Quick start

**1. Install foundryctl**

Download from [GitHub Releases](https://github.com/signoz/foundry/releases), or build from source:

```bash
git clone https://github.com/signoz/foundry.git && cd foundry
go build -o foundryctl ./cmd/foundryctl
```

**2. Create a Casting**

Create `casting.yaml`:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
```

**3. Deploy**

```bash
foundryctl cast -f casting.yaml
```

## The Foundry Model

Foundry uses a metalworking metaphor: you define a **Casting**, which contains **Moldings** (components), and Foundry **forges** them into **Pours** (generated files).

### Casting

A **Casting** is a complete SigNoz deployment definition. The configuration uses a Kubernetes-like YAML syntax:

```yaml
apiVersion: v1alpha1
metadata:
  name: <deployment-name>
spec:
  deployment:
    platform: <platform>    # Optional: render, aws, gcp, etc.
    mode: <mode>            # Required: docker, systemd
    flavor: <flavor>        # Required: compose, binary, blueprint
  
  # Molding configurations (all optional - defaults provided)
  signoz:
    spec:
      image: <image>
  
  telemetrystore:
    spec:
      image: <image>
  
  telemetrykeeper:
    spec:
      image: <image>
  
  metastore:
    spec:
      image: <image>
  
  ingester:
    spec:
      image: <image>
```

### Supported deployments

| Deployment | Example |
|------------|---------|
| Docker Compose | [examples/docker/compose/](examples/docker/compose/) |
| Systemd (binary) | [examples/systemd/binary/](examples/systemd/binary/) |
| Render Blueprint | [examples/render/blueprint/](examples/render/blueprint/) |

### Moldings

**Moldings** are the individual components that make up a SigNoz deployment:

| Molding | Implementation |
|---------|----------------|
| **TelemetryStore** | ClickHouse |
| **TelemetryKeeper** | ClickHouse Keeper |
| **MetaStore** | PostgreSQL, SQLite |
| **Ingester** | SigNoz OTel Collector |
| **SigNoz** | SigNoz |

### Pours

**Pours** are the generated deployment and configuration files. When you run `forge`, Foundry creates the `pours/` directory containing everything needed to run SigNoz.

```
pours/
└── deployment/
    ├── compose.yaml
    └── configs/
        ├── ingester/
        │   ├── ingester.yaml
        │   └── opamp.yaml
        ├── telemetrykeeper/
        │   └── keeper-0.yaml
        └── telemetrystore/
            ├── config.yaml
            └── functions.yaml
```

## CLI reference

```
Usage:
  foundryctl [command]

Available Commands:
  gauge       Gauge whether required tools are available
  forge       Forge configuration and deployment files
  cast        Cast to the target environment
  gen         Generate example files for all supported deployments
  help        Help about any command

Flags:
  -d, --debug          Enable debug mode
  -f, --file string    Path to the Casting configuration file (default "casting.yaml")
  -p, --pours string   Directory for Pours (default "./pours")
  -h, --help           Help for foundryctl
```

### gauge

Validates that all required tools are installed for your deployment mode:

```bash
foundryctl gauge -f casting.yaml
```

### forge

Generates deployment and configuration files based on your Casting:

```bash
foundryctl forge -f casting.yaml -p ./pours
```

### cast

Deploys SigNoz to your target environment. Runs `gauge` and `forge` automatically unless skipped:

```bash
foundryctl cast -f casting.yaml

# Skip gauge check
foundryctl cast --no-gauge

# Skip forge (use existing Pours)
foundryctl cast --no-forge
```

### gen

Generates example Casting configurations for all supported deployment modes:

```bash
foundryctl gen
```

## What's next

- Explore the [example configurations](examples/) for different deployment scenarios
- Read the [SigNoz documentation](https://signoz.io/docs/) to learn more about SigNoz
- Join the [SigNoz community on Slack](https://signoz.io/slack) to get help

## How can I get help?

- **Issues**: [GitHub Issues](https://github.com/signoz/foundry/issues)
- **Documentation**: [SigNoz Docs](https://signoz.io/docs/)
- **Community**: [SigNoz Slack](https://signoz.io/slack)
