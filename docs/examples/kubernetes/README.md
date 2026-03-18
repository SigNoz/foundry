# Kubernetes

| Example | Description |
|---|---|
| [kustomize](kustomize/) | Deployment with all Foundry defaults |
| [kustomize-patches](kustomize-patches/) | Deployment with custom storage, resources, scheduling, and networking |

## Prerequisites

- Kubernetes cluster (1.24+)
- `kubectl` with kustomize support

## Usage

```bash
foundryctl cast -f casting.yaml
```

or

```bash
foundryctl forge -f casting.yaml
```

To deploy:

```bash
kubectl apply -k pours/deployment/
```

## Generated Output

Foundry generates per-component directories under `pours/deployment/`. Each directory contains Kubernetes manifests for that component. Patch targets reference files relative to `pours/`.

```text
pours/deployment/
  kustomization.yaml                # root kustomization
  namespace.yaml                    # namespace definition
  signoz/                           # SigNoz UI + API
    statefulset.yaml                  # SigNoz StatefulSet
    service.yaml                      # SigNoz Service (ClusterIP)
    serviceaccount.yaml
    kustomization.yaml
  ingester/                         # OTel Collector (ingestion + processing)
    deployment.yaml                   # Ingester Deployment
    service.yaml                      # Ingester Service
    configmap.yaml                    # Collector config
    serviceaccount.yaml
    kustomization.yaml
  telemetrystore/                   # ClickHouse
    clickhouse/
      clickhouseinstallation.yaml     # ClickHouse CR (volumes, resources, config)
      configmap.yaml                  # custom functions
      kustomization.yaml
  clickhouse-operator/
    deployment.yaml                 # operator Deployment
    clusterrole.yaml
    clusterrolebinding.yaml
    configmap.yaml
    service.yaml
    serviceaccount.yaml
    kustomization.yaml
  telemetrykeeper/
    clickhousekeeper/                  # ClickHouse Keeper
      clickhousekeeperinstallation.yaml # Keeper CR
      kustomization.yaml
  metastore/                        # PostgreSQL metadata store
    postgres/
      statefulset.yaml                # PostgreSQL StatefulSet
      service.yaml
      serviceaccount.yaml
      kustomization.yaml
  telemetrystore-migrator/          # schema migration job
    job.yaml
    kustomization.yaml
```
