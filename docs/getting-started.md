# Getting Started

Install `foundryctl` and deploy SigNoz in three steps.

## 1. Install foundryctl

Download the latest release from [GitHub Releases](https://github.com/signoz/foundry/releases), or use the commands below.

**Linux:**

```bash
curl -L "https://github.com/SigNoz/foundry/releases/latest/download/foundry_linux_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g').tar.gz" -o foundry.tar.gz
tar -xzf foundry.tar.gz
cd foundry_linux_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/arm64/arm64/g')
```

**macOS:**

```bash
curl -L "https://github.com/SigNoz/foundry/releases/latest/download/foundry_darwin_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/arm64/arm64/g').tar.gz" -o foundry.tar.gz
tar -xzf foundry.tar.gz
cd foundry_darwin_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/arm64/arm64/g')
```

**Windows (PowerShell):**

```powershell
$ARCH = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
Invoke-WebRequest -Uri "https://github.com/SigNoz/foundry/releases/latest/download/foundry_windows_${ARCH}.tar.gz" -OutFile foundry.tar.gz -UseBasicParsing
tar -xzf foundry.tar.gz
cd foundry_windows_$ARCH
```

The archive extracts to a directory named `foundry_<os>_<arch>` (for example, `foundry_darwin_arm64`). The `cd` command above moves you into it.

Verify the installation:

```bash
./bin/foundryctl --help
```

## 2. Create a casting

A casting is a YAML file that describes your SigNoz deployment. Create a file called `casting.yaml`:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
```

This minimal casting deploys SigNoz using Docker Compose with all default settings.

> [!TIP]
> Run `foundryctl gen examples` to generate working casting files for every supported deployment mode (Docker, Kubernetes, systemd, Render, and more).

## 3. Deploy

```bash
./bin/foundryctl cast -f casting.yaml
```

Foundry validates your tools (`gauge`), generates deployment files (`forge`), and deploys SigNoz (`cast`) in one step.

## Validate

Check that SigNoz is running:

```bash
docker ps
```

All containers should show `Up` status. Open `http://localhost:8080` to access the SigNoz UI.

## What's next

- [Casting concepts](concepts/casting.md) - understand casting files in depth
- [Moldings](concepts/moldings.md) - configure individual components
- [Patches](concepts/patches.md) - customize generated output
- [CLI reference](reference/cli.md) - all commands and flags
- [Examples](examples/) - working examples for Docker, Kubernetes, systemd, and cloud platforms
