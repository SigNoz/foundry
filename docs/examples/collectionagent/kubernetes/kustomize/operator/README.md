# Kubernetes Collection Agent (OpenTelemetry Operator)

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |
| **Annotation** | `foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator` |

## Overview

Deploys the SigNoz Collection Agents on a Kubernetes cluster as
`OpenTelemetryCollector` resources reconciled by the OpenTelemetry Operator,
composed by kustomize and exporting to any SigNoz: Self-Hosted Community,
Self-Hosted Enterprise, or SigNoz Cloud.

This is the [`kustomize` casting](../README.md) with one annotation stated.
The collectors, their receivers and their config are the same. The
difference is who owns the workload: foundry pours one
`OpenTelemetryCollector` per collector kind, and the operator turns it into
the workload, its Service and its ConfigMap.

Kubernetes telemetry comes from two collectors with different scopes, one
casting each:

- `agent/` runs on every node, `spec.mode: daemonset`:
  - Pod and container metrics from the kubelet through the `kubeletstats`
    receiver
  - Node metrics from the mounted host filesystem through the `hostmetrics`
    receiver
  - Pod logs from `/var/log/pods` through the `filelog` receiver
  - OTLP intake for your applications on host ports 4317 (gRPC) and 4318
    (HTTP)
- `deployment/` runs once per cluster, `spec.mode: deployment`:
  - Cluster metrics (workload status, pod phase, node conditions,
    allocatables) from the API server through the `k8s_cluster` receiver
  - Kubernetes events as logs through the `k8s_events` receiver

Run both. The SigNoz Kubernetes views require both sources: entity
resolution breaks when either the kubelet metrics or the cluster-level
metrics are missing. Both castings share the `metadata.name`, so they compose
into one namespace: `<metadata.name>`, or the namespace the
`foundry.signoz.io/kubernetes-namespace` annotation names.

The collector config is inlined in the resource's `spec.config`, so no
ConfigMap or config file is poured, and the operator rolls the workload when
the config changes. Foundry pours the service account and the RBAC each
collector needs, and names that service account in `spec.serviceAccount`.

## Prerequisites

- A Kubernetes cluster and `kubectl` configured against it
- A running SigNoz to receive the telemetry
- The OpenTelemetry Operator v0.139.0, installed once per cluster. The
  version matches the `otel/opentelemetry-collector-contrib:0.139.0` collector
  image foundry directs. The operator itself needs cert-manager for its
  admission webhooks, so install that first:

  ```bash
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
  kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.139.0/opentelemetry-operator.yaml
  kubectl get deployment opentelemetry-operator-controller-manager -n opentelemetry-operator-system
  ```

Foundry never installs cert-manager, the operator or its CRDs. Without the
operator, `foundryctl cast` fails on the `OpenTelemetryCollector` resource with
kubectl's own "no matches for kind" error. See the SigNoz guides to [install the operator](https://signoz.io/docs/opentelemetry-collection-agents/k8s/otel-operator/install/)
and to [configure collectors with it](https://signoz.io/docs/opentelemetry-collection-agents/k8s/otel-operator/configure/).

## Configuration

The node agent (`agent/casting.yaml`):

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator
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
  annotations:
    foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator
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
`pours/collectionagent/collector/<kind>/`: `kustomization.yaml`, the
namespace, RBAC, and `opentelemetrycollector.yaml` with the collector config
inline. A casting file declaring both kinds pours a root per document, and
casting applies each.

## Cast

```bash
foundryctl cast -f agent/casting.yaml -p agent/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

This runs `kubectl apply -k` on the collector root using your current
kubeconfig context. To inspect what would be applied first:

```bash
kubectl kustomize agent/pours/collectionagent/collector/agent
```

The operator names what it creates after the resource:
`<metadata.name>-collector-agent-collector` for the DaemonSet and
`<metadata.name>-collector-deployment-collector` for the Deployment. To check
them:

```bash
kubectl get opentelemetrycollectors,daemonsets,deployments -n signoz
```

## Customization

Override any collector setting through `spec.collector.spec.config.data`; user
keys win over generated ones, and the result lands in the resource's
`spec.config`. For changes to the generated `opentelemetrycollector.yaml`
itself (other `OpenTelemetryCollector` fields, resource limits, node
selectors), use [patches](../../../../../concepts/patches.md).

## Annotations

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-collector-controller` | `default` | Controller that owns the collector workload: `default` pours the DaemonSet or Deployment for the built-in Kubernetes controllers, `opentelemetry-operator` pours an `OpenTelemetryCollector` resource for the operator to reconcile |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the collector is deployed into |

Example deploying into a namespace of its own:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator
    foundry.signoz.io/kubernetes-namespace: observability
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
