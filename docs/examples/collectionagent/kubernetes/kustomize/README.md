# Kubernetes Collection Agent

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |

## Overview

Deploys the SigNoz Collection Agents on a Kubernetes cluster as plain
manifests composed by kustomize, exporting to any SigNoz: Self-Hosted
Community, Self-Hosted Enterprise, or SigNoz Cloud.

Kubernetes telemetry comes from two collectors with different scopes, one
casting each:

- `agent/` runs on every node as a DaemonSet:
  - Pod and container metrics from the kubelet through the `kubeletstats`
    receiver
  - Node metrics from the mounted host filesystem through the `hostmetrics`
    receiver
  - Pod logs from `/var/log/pods` through the `filelog` receiver
  - OTLP intake for your applications on host ports 4317 (gRPC) and 4318
    (HTTP), also reachable node-locally as a service
- `deployment/` runs once per cluster as a Deployment:
  - Cluster metrics (workload status, pod phase, node conditions,
    allocatables) from the API server through the `k8s_cluster` receiver
  - Kubernetes events as logs through the `k8s_events` receiver

Run both. The SigNoz Kubernetes views require both sources: entity
resolution breaks when either the kubelet metrics or the cluster-level
metrics are missing. Together they match what the SigNoz k8s-infra chart
deploys. Both castings share the `metadata.name`, so they compose into one
namespace.

Kubernetes metadata lands on every signal through the `k8sattributes`
processor, and the collector config enters the cluster through a
`configMapGenerator`: a config change re-hashes the ConfigMap name and rolls
the workload on the next apply.

## Prerequisites

- A Kubernetes cluster and `kubectl` configured against it
- A running SigNoz to receive the telemetry

## Configuration

The node agent (`agent/casting.yaml`):

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
        K8S_CLUSTER_NAME: <cluster-name>
```

The cluster collector (`deployment/casting.yaml`):

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
        K8S_CLUSTER_NAME: <cluster-name>
```

`K8S_CLUSTER_NAME` names the cluster on every signal as `k8s.cluster.name`.
Set it: the SigNoz Kubernetes views require it on every entity, and queries
fail silently without it.

## Forge

```bash
foundryctl forge -f agent/casting.yaml -p agent/pours
foundryctl forge -f deployment/casting.yaml -p deployment/pours
```

Each collector kind pours a kustomize root of its own under
`<kind>/pours/collectionagent/collector/<kind>/`: `kustomization.yaml`, the
namespace, RBAC, service, the workload manifest, and the collector config the
configMapGenerator reads. A casting file declaring both kinds pours a root per
document, and casting applies each.

## Cast

```bash
foundryctl cast -f agent/casting.yaml -p agent/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

This runs `kubectl apply -k` against the pours using your current kubeconfig
context. To inspect what would be applied first:

```bash
kubectl kustomize agent/pours/collectionagent/collector/agent
```
