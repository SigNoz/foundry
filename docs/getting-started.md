# Getting Started

Install `foundryctl` and deploy SigNoz in three steps.

## 1. Install foundryctl

The fastest path is the install script, which detects your OS/arch, verifies the
download checksum, installs the binary into `$XDG_BIN_HOME` or `~/.local/bin`,
and prints a PATH hint if needed:

```bash
curl -fsSL https://signoz.io/foundry.sh | bash
```

Pin a specific version:

```bash
curl -fsSL https://signoz.io/foundry.sh | FOUNDRY_VERSION=v0.1.4 bash
```

### Manual install

If you prefer to download the release archive yourself:

**Linux:**

```bash
curl -L "https://github.com/SigNoz/foundry/releases/latest/download/foundry_linux_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g').tar.gz" -o foundry.tar.gz
tar -xzf foundry.tar.gz
```

**macOS:**

```bash
curl -L "https://github.com/SigNoz/foundry/releases/latest/download/foundry_darwin_$(uname -m | sed 's/x86_64/amd64/g' | sed 's/arm64/arm64/g').tar.gz" -o foundry.tar.gz
tar -xzf foundry.tar.gz
```

**Windows (PowerShell):**

```powershell
$ARCH = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
Invoke-WebRequest -Uri "https://github.com/SigNoz/foundry/releases/latest/download/foundry_windows_${ARCH}.tar.gz" -OutFile foundry.tar.gz -UseBasicParsing
tar -xzf foundry.tar.gz
```

The archive extracts to a directory named `foundry_<os>_<arch>` (for example, `foundry_darwin_arm64`) containing `bin/foundryctl`.

### Add to PATH

To run `foundryctl` from anywhere, move the binary onto your `PATH`:

```bash
mkdir -p "$HOME/.local/bin"
mv foundry_*/bin/foundryctl "$HOME/.local/bin/"
```

If `~/.local/bin` isn't already on your `PATH` (common on macOS), add it to your shell config:

```bash
# zsh (~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"

# bash (~/.bashrc on Linux, ~/.bash_profile on macOS)
export PATH="$HOME/.local/bin:$PATH"

# fish (~/.config/fish/config.fish)
fish_add_path $HOME/.local/bin
```

Reload your shell or `source` the file you edited.

### Verify

```bash
foundryctl --help
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
