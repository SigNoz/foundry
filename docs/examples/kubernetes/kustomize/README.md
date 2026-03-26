# Kubernetes with Kustomize

| Field | Value |
| --- | --- |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |
| **Platform** | `-` |

## Overview

Deploys SigNoz on Kubernetes using Kustomize. Foundry generates per-component directories with Kubernetes manifests and a root `kustomization.yaml`.

## Prerequisites

- Kubernetes cluster (1.24+)
- `kubectl` with kustomize support

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: kustomize
    mode: kubernetes
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
# Generate manifests
foundryctl forge -f casting.yaml

# Apply with kubectl
kubectl apply -k pours/deployment/
```

## Generated output

```text
pours/deployment/
  kustomization.yaml
  namespace.yaml
  signoz/
    statefulset.yaml
    service.yaml
    serviceaccount.yaml
    kustomization.yaml
  ingester/
    deployment.yaml
    service.yaml
    configmap.yaml
    serviceaccount.yaml
    kustomization.yaml
  telemetrystore/
    clickhouse/
      clickhouseinstallation.yaml
      configmap.yaml
      kustomization.yaml
  clickhouse-operator/
    deployment.yaml
    clusterrole.yaml
    clusterrolebinding.yaml
    configmap.yaml
    service.yaml
    serviceaccount.yaml
    kustomization.yaml
  telemetrykeeper/
    clickhousekeeper/
      clickhousekeeperinstallation.yaml
      kustomization.yaml
  metastore/
    postgres/
      statefulset.yaml
      service.yaml
      serviceaccount.yaml
      kustomization.yaml
  telemetrystore-migrator/
    job.yaml
    kustomization.yaml
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

To set resource limits, storage classes, or scheduling constraints on the generated manifests, use [patches](../../concepts/patches.md). See the [kustomize-patches](../kustomize-patches/) example for a complete working configuration.
