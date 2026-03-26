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

# Apply CRDs first (cast does this automatically)
kubectl apply -f https://raw.githubusercontent.com/Altinity/clickhouse-operator/0.25.3/deploy/operatorhub/0.25.3/clickhouseinstallations.clickhouse.altinity.com.crd.yaml
kubectl apply -f https://raw.githubusercontent.com/Altinity/clickhouse-operator/0.25.3/deploy/operatorhub/0.25.3/clickhouseinstallationtemplates.clickhouse.altinity.com.crd.yaml
kubectl apply -f https://raw.githubusercontent.com/Altinity/clickhouse-operator/0.25.3/deploy/operatorhub/0.25.3/clickhouseoperatorconfigurations.clickhouse.altinity.com.crd.yaml
kubectl apply -f https://raw.githubusercontent.com/Altinity/clickhouse-operator/0.25.3/deploy/operatorhub/0.25.3/clickhousekeeperinstallations.clickhouse-keeper.altinity.com.crd.yaml

# Apply with kubectl
kubectl apply -k pours/deployment/
```

> [!NOTE]
> `foundryctl cast` automatically fetches and applies the four Altinity ClickHouse Operator CRDs (v0.25.3) from GitHub before running `kubectl apply -k`. If you apply manually, you must install the CRDs first or the ClickHouseInstallation and ClickHouseKeeperInstallation resources will fail to create.

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

### Native Kustomize patches

Since Foundry generates standard Kustomize bases, you can also use native Kustomize patches on the generated `kustomization.yaml`. This lets you use strategic merge patches or overlays for environment-specific customization without re-forging.

Use a Foundry patch to inject a `patches` block into the root `kustomization.yaml`:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: kustomize
    mode: kubernetes
  patches:
    - target: "deployment/kustomization.yaml"
      operations:
        - op: add
          path: /patches
          value:
            - target:
                kind: StatefulSet
                name: signoz-signoz
              patch: |-
                apiVersion: apps/v1
                kind: StatefulSet
                metadata:
                  name: signoz-signoz
                spec:
                  template:
                    spec:
                      nodeSelector:
                        node-role.kubernetes.io/observability: ""
```

Or create an overlay directory that references the generated base:

```
my-deployment/
├── base/                    # Copy of pours/deployment/
│   └── ...
└── overlays/
    └── prod/
        ├── kustomization.yaml
        └── increase-resources.yaml
```

```yaml
# overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../base
patches:
- path: increase-resources.yaml
  target:
    kind: StatefulSet
    name: signoz-clickhouse
```

```bash
kubectl apply -k overlays/prod/
```
