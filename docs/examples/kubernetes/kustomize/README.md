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

# Operators first (CRDs + clickhouse-operator), then the workloads
kubectl apply -k pours/deployment/operators/clickhouse-operator
kubectl wait --for=condition=Established crd/clickhouseinstallations.clickhouse.altinity.com crd/clickhousekeeperinstallations.clickhouse-keeper.altinity.com
kubectl apply -k pours/deployment/
```

> [!NOTE]
> `pours/deployment/kustomization.yaml` covers the namespace and the SigNoz components only. `operators/clickhouse-operator` is a separate tier: it lists the four Altinity CRDs (v0.25.3) as remote resources, so applying it needs network access to raw.githubusercontent.com, and it has to be applied and established before the root, which is what `foundryctl cast` does.

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
  operators/
    clickhouse-operator/
      namespace.yaml
      deployment.yaml
      clusterrole.yaml
      clusterrolebinding.yaml
      configmap.yaml
      service.yaml
      serviceaccount.yaml
      kustomization.yaml
  telemetrystore/
    clickhouse/
      clickhouseinstallation.yaml
      configmap.yaml
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

## MCP server (optional)

Foundry can deploy the [SigNoz MCP server](https://github.com/SigNoz/signoz-mcp-server) alongside the stack so AI clients can query your telemetry. It is disabled by default; enable it in the casting:

```yaml
apiVersion: v1alpha1
kind: Installation
metadata:
  name: signoz
spec:
  deployment:
    flavor: kustomize
    mode: kubernetes
  mcp:
    spec:
      enabled: true
```

This adds a `signoz-mcp` deployment and a ClusterIP service on port `8000`; the in-cluster endpoint is `http://signoz-mcp.signoz:8000/mcp`. Expose it outside the cluster per your setup.

```bash
kubectl -n signoz get pods -l app.kubernetes.io/component=mcp
```

To connect an AI client (mint an API key, configure Claude Code or Claude Desktop), see [MCP server](../../../concepts/mcp-server.md).

## Customization

To set resource limits, storage classes, or scheduling constraints on the generated manifests, use [patches](../../../concepts/patches.md). See the [kustomize-patches](../kustomize-patches/) example for a complete working configuration.

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
