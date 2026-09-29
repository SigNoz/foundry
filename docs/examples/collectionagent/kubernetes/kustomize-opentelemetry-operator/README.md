# Kubernetes Collection Agent (OpenTelemetry Operator)

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |
| **Annotation** | `foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator` |

## Overview

Deploys the SigNoz Collection Agents as two `OpenTelemetryCollector`
resources that the OpenTelemetry Operator runs: a node agent on every node
and a cluster collector once per cluster, both exporting to SigNoz. Cast
both. For what each collector collects, see the
[`kustomize` casting](../kustomize/README.md).

## Prerequisites

- A Kubernetes cluster and `kubectl` configured against it
- A running SigNoz to receive the telemetry
- The OpenTelemetry Operator v0.139.0 and cert-manager. Install cert-manager
  first, then the operator:

  ```bash
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
  kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/download/v0.139.0/opentelemetry-operator.yaml
  kubectl get deployment opentelemetry-operator-controller-manager -n opentelemetry-operator-system
  ```

See the SigNoz guides to [install the operator](https://signoz.io/docs/opentelemetry-collection-agents/k8s/otel-operator/install/)
and to [configure collectors with it](https://signoz.io/docs/opentelemetry-collection-agents/k8s/otel-operator/configure/).

## Configuration

The node agent:

```yaml
# agent/casting.yaml
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

The cluster collector:

```yaml
# deployment/casting.yaml
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

In both files, set `SIGNOZ_INGESTION_ENDPOINT` to your SigNoz and
`K8S_CLUSTER_NAME` to the name the cluster shows under in the SigNoz
Kubernetes views.

## Forge

```bash
foundryctl forge -f agent/casting.yaml -p agent/pours
foundryctl forge -f deployment/casting.yaml -p deployment/pours
```

The pours land in `<kind>/pours/collectionagent/collector/<kind>/`, each
holding the collector's `opentelemetrycollector.yaml`.

## Cast

```bash
foundryctl cast -f agent/casting.yaml -p agent/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

To inspect what would be applied first:

```bash
kubectl kustomize agent/pours/collectionagent/collector/agent
```

## Validate

```bash
kubectl get opentelemetrycollectors,daemonsets,deployments -n signoz
```

The operator creates the `signoz-collector-agent-collector` DaemonSet and the
`signoz-collector-deployment-collector` Deployment.

## Customization

Change collector settings through `spec.collector.spec.config.data`. For any
other field of the `OpenTelemetryCollector` resource, such as resource limits
or node selectors, use [patches](../../../../concepts/patches.md).

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
