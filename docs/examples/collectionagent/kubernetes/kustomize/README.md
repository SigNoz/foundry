# Kubernetes Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |

## Overview

Deploys a SigNoz Collection Agent on a Kubernetes cluster as plain manifests
composed by kustomize. The agent runs the OpenTelemetry Collector on every
node as a DaemonSet, receives OTLP from your applications on host ports 4317
(gRPC) and 4318 (HTTP), and exports to any SigNoz: Self-Hosted Community,
Self-Hosted Enterprise, or SigNoz Cloud.

The collector config enters the cluster through a `configMapGenerator`, so a
config change re-hashes the ConfigMap name and rolls the workload on the next
apply.

## Prerequisites

- A Kubernetes cluster and `kubectl` configured against it
- A running SigNoz to receive the telemetry

## Configuration

The default casting (this directory's `casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    mode: kubernetes
    flavor: kustomize
  collector:
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
```

## Forge

```bash
foundryctl forge -f casting.yaml
```

The pours land in `pours/collectionagent/`: `kustomization.yaml`,
`namespace.yaml`, `serviceaccount.yaml`, the workload manifest, and the
collector config under `collector/`.

## Cast

```bash
foundryctl cast -f casting.yaml
```

This runs `kubectl apply -k` against the pours using your current kubeconfig
context. To inspect what would be applied first:

```bash
kubectl kustomize pours/collectionagent
```

## Collector kinds

`spec.collector.kind` selects the workload scope. `agent` (the default) runs
a DaemonSet with host ports. `deployment` (environment-scoped collection)
runs a Deployment behind a ClusterIP service.

An environment-scoped collector:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
spec:
  deployment:
    mode: kubernetes
    flavor: kustomize
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
```

## Status

Every kind ships the base collector config: OTLP intake, batching, memory
protection, and export to SigNoz. The Kubernetes telemetry collection (node,
pod, and cluster signals) lands next.
