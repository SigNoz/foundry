# Kubernetes with Helm

| Field | Value |
| --- | --- |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |
| **Platform** | `-` |

## Overview

Generates a `values.yaml` for the SigNoz Helm chart. Foundry translates the casting spec into Helm values so you can deploy with a single `helm install`.

## Prerequisites

- Kubernetes cluster (1.24+)
- Helm 3.x
- SigNoz Helm chart repo (`https://charts.signoz.io`)

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Generate values.yaml
foundryctl forge -f casting.yaml

# Add the SigNoz Helm repo
helm repo add signoz https://charts.signoz.io

# Install with Helm
helm install signoz signoz/signoz -f pours/deployment/values.yaml -n signoz --create-namespace
```

> [!NOTE]
> `foundryctl cast` is idempotent. It detects whether a Helm release already exists and runs `helm upgrade` instead of `helm install` accordingly.

## Generated output

```text
pours/deployment/
  values.yaml
```

## After deployment

```bash
# Check pod status
kubectl get pods -n signoz

# Port-forward the SigNoz UI
kubectl port-forward svc/signoz -n signoz 8080:8080
```

Open `http://localhost:8080` to access the SigNoz UI.

## Customization

To set resource limits, storage classes, or scheduling constraints on the generated `values.yaml`, use [patches](../../concepts/patches.md). See the [helm-patches](../helm-patches/) example for a complete working configuration.

## Annotations

Optional annotations to override the default Helm chart source. These are not required for standard deployments.

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-helm-casting-chart` | `signoz/signoz` | Helm chart reference |
| `foundry.signoz.io/kubernetes-helm-casting-repo-url` | `https://charts.signoz.io` | Helm chart repository URL |
| `foundry.signoz.io/kubernetes-helm-casting-repo-name` | `signoz` | Helm chart repository name |
| `foundry.signoz.io/kubernetes-helm-casting-forge-chart` | - | Set to `true` to download and bundle the chart locally during forge |

Example with a custom chart repo:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-helm-casting-repo-url: https://my-registry.example.com/charts
    foundry.signoz.io/kubernetes-helm-casting-chart: my-registry/signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
```
