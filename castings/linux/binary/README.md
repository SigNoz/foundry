# Standard Linux Binary Casting

This directory contains the **standard casting** for running SigNoz on **Linux** using **Native Binaries** and **Systemd**.

A casting is a fully assembled deployment bundle. Unlike Docker, this method runs services directly on the host operating system.

## 1. Pours (Generated Configuration)

This casting does not use static configuration files. Instead, it generates a custom set of configurations called **Pours**.

These generated files will appear in:

```bash
./pours/
```

This directory acts as the "Staging Area". You can inspect the files in this directory to verify the configuration (ports, paths, hostnames) before applying them to your system.

## 2. Environment Variables

To generate the pours, you must define your environment in a `.env` file:

```bash
./.env
```

A minimal example:

```env
# Versions
SIGNOZ_VERSION=v0.46.0
OTEL_VERSION=0.102.7

# Installation Paths (Where binaries and data go)
SIGNOZ_INSTALL_DIR=/opt/signoz
SIGNOZ_DATA_DIR=/var/lib/signoz

# Network Configuration
CLICKHOUSE_HOST=127.0.0.1
CLICKHOUSE_PORT=9000
```

Adjust these values to match your server architecture (e.g., if ClickHouse is on a different node, update `CLICKHOUSE_HOST`).

## 3. Forge: Generate the Pours

Run the forge command to generate your configurations:

```bash
./setup.sh forge
```

This will:
* Load variables from your `.env` file
* Generate component configurations (ClickHouse, Zookeeper, SigNoz)
* Generate Systemd service units with correct paths
* Write everything into the `./pours/` directory

**Tip:** Always inspect the `./pours/` directory after forging to ensure the configuration looks correct.

## 4. Cast: Run the Deployment

Once the pours are generated, apply them to the system using the cast command:

```bash
sudo ./setup.sh cast
```

**Note:** This command requires `root` privileges.

This will:
1.  Download necessary binaries (if missing)
2.  Create system users (`signoz`, `zookeeper`)
3.  Install the configurations from `./pours/` to `/etc/`
4.  Enable and start the Systemd services