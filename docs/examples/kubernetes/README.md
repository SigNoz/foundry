# Kubernetes

Foundry supports multiple deployment flavors for Kubernetes. Each flavor generates different output formats suited to different workflows.

## Supported Castings

| Flavor | Mode | Description | Examples |
|---|---|---|---|
| `kustomize` | `kubernetes` | Generates per-component directories with Kubernetes manifests and Kustomize overlays | [kustomize](kustomize/), [kustomize-patches](kustomize-patches/) |
| `helm` | `kubernetes` | Generates a single `values.yaml` for the SigNoz Helm chart | [helm](helm/), [helm-patches](helm-patches/) |

## Prerequisites

- Kubernetes cluster (1.24+)

### Kustomize

- `kubectl` with kustomize support

### Helm

- Helm 3.x
- SigNoz Helm chart repo (`https://charts.signoz.io`)

## Usage

```bash
# Generate deployment files
foundryctl forge -f casting.yaml

# Full pipeline: gauge → forge → deploy
foundryctl cast -f casting.yaml
```

---

## Kustomize

### casting.yaml

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: kustomize
    mode: kubernetes
```

### Deploy

```bash
kubectl apply -k pours/deployment/
```

### Generated Output

Foundry generates per-component directories under `pours/deployment/`. Each directory contains Kubernetes manifests for that component. Patch targets reference files relative to `pours/`.

```text
pours/deployment/
  kustomization.yaml                # root kustomization
  namespace.yaml                    # namespace definition
  signoz/                           # SigNoz UI + API
    statefulset.yaml
    service.yaml
    serviceaccount.yaml
    kustomization.yaml
  ingester/                         # OTel Collector (ingestion + processing)
    deployment.yaml
    service.yaml
    configmap.yaml
    serviceaccount.yaml
    kustomization.yaml
  telemetrystore/                   # ClickHouse
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
    clickhousekeeper/               # ClickHouse Keeper
      clickhousekeeperinstallation.yaml
      kustomization.yaml
  metastore/                        # PostgreSQL metadata store
    postgres/
      statefulset.yaml
      service.yaml
      serviceaccount.yaml
      kustomization.yaml
  telemetrystore-migrator/          # schema migration job
    job.yaml
    kustomization.yaml
```

---

## Helm

### casting.yaml

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
```

### Deploy

```bash
# Using forge + helm install separately
foundryctl forge -f casting.yaml
helm install signoz signoz/signoz -f pours/deployment/values.yaml -n signoz --create-namespace

# Or using cast (forge + deploy in one step)
foundryctl cast -f casting.yaml
```

### Generated Output

Foundry generates a single Helm values file under `pours/deployment/`. Patch targets reference this file relative to `pours/`.

```text
pours/deployment/
  values.yaml                       # Helm chart values
```

The generated `values.yaml` configures the SigNoz Helm chart with component images, replicas, environment variables, and cluster settings derived from the casting spec.
