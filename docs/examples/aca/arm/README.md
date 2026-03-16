# Azure Container Apps (ACA) Casting

This guide explains how to use the ACA casting for deploying SigNoz on Azure Container Apps.

## Prerequisites

- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and authenticated (`az login`)
- An Azure Container Apps environment
- A resource group with the ACA environment provisioned
- `foundryctl` binary

### Azure Resources

Before running `foundryctl cast`, ensure the following Azure resources exist:

1. **Resource Group** : contains all deployed resources
2. **Container Apps Environment** : the ACA environment where apps are deployed

## Casting Configuration

Create a `casting.yaml` file:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/aca-resource-group: <resource-group-name>
    foundry.signoz.io/aca-environment: <aca-environment-name>
    foundry.signoz.io/aca-subscription-id: <azure-subscription-id>
spec:
  deployment:
    flavor: arm
    platform: aca
```

### Required Annotations

| Annotation | Description |
|---|---|
| `foundry.signoz.io/aca-resource-group` | Azure resource group name |
| `foundry.signoz.io/aca-environment` | Container Apps environment name |
| `foundry.signoz.io/aca-subscription-id` | Azure subscription ID |

## Deployment

### 1. Generate Deployment Files

```bash
foundryctl forge -f casting.yaml
```

This generates container app YAML specs under `pours/deployment/`. Component configs (ClickHouse, OTel Collector, etc.) are embedded as secrets in the container app definitions — no external storage is needed.

### 2. Deploy to Azure

```bash
foundryctl cast -f casting.yaml
```


### 3. Verify Deployment

```bash
az containerapp list --resource-group <resource-group-name> --output table
```

View logs for a specific app:

```bash
az containerapp logs show --name signoz-signoz --resource-group <resource-group-name> --follow
```

## Architecture

Each SigNoz component is deployed as a separate Container App. Configuration files are delivered via ACA secret volumes — the config content is embedded as secrets in the container app YAML and mounted as files at runtime.

### Per-Component Knobs

You can customize resource allocation and scaling per component:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/aca-resource-group: my-rg
    foundry.signoz.io/aca-environment: my-env
    foundry.signoz.io/aca-subscription-id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
spec:
  deployment:
    flavor: arm
    platform: aca
  telemetryStore:
    spec:
      config:
        knobs:
          resources:
            cpu: 4
            memory: 8Gi
          scale:
            minReplicas: 1
            maxReplicas: 3
```

Available knobs for all components (`telemetryStore`, `telemetryKeeper`, `signoz`, `ingester`):

| Knob | Description | Default |
|---|---|---|
| `resources.cpu` | CPU allocation | Varies per component |
| `resources.memory` | Memory allocation | Varies per component |