# Kubernetes Collection Agent with Helm (OpenTelemetry Operator)

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |
| **Annotation** | `foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator` |

## Overview

Deploys the SigNoz Collection Agents as two `OpenTelemetryCollector`
resources that the OpenTelemetry Operator runs: a node agent on every node
and a cluster collector once per cluster, both exporting to SigNoz. Cast
both. For what each collector collects, see the
[`helm` casting](../helm/README.md).

Under the annotation the casting installs the upstream
`opentelemetry-kube-stack` chart, version 0.13.0, with its operator subchart,
its CRDs and its cleanup Job turned off. Each release renders one
`OpenTelemetryCollector` resource, one ClusterRole
`<release>-collector` and one ClusterRoleBinding to the service account the
operator creates. The collector image comes from
`spec.collector.spec.image`, so the collector runs 0.139.0; the chart's
appVersion (0.141.0) names the operator subchart, which is turned off.

Each kind installs as its own Helm release, `<metadata.name>-collector-agent`
and `<metadata.name>-collector-deployment`, and the operator creates the
`<release>-collector` workload.

## Prerequisites

- A Kubernetes cluster and a kubeconfig context pointing at it
- Helm 3.x
- A running SigNoz to receive the telemetry
- The OpenTelemetry Operator v0.139.0 and cert-manager. Install cert-manager
  first, then the operator with its Helm chart, version 0.100.0, which installs
  operator v0.139.0 and its CRDs:

  ```bash
    helm repo add jetstack https://charts.jetstack.io
    helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
    helm repo update
    helm install cert-manager jetstack/cert-manager \
      --create-namespace --namespace cert-manager --set crds.enabled=true
    helm install opentelemetry-operator open-telemetry/opentelemetry-operator --version 0.100.0 \
      --create-namespace --namespace opentelemetry-operator-system
    helm status opentelemetry-operator -n opentelemetry-operator-system
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
    flavor: helm
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
    flavor: helm
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <cluster-name>
```

In both files, set `SIGNOZ_INGESTION_ENDPOINT` to your SigNoz and
`K8S_CLUSTER_NAME` to the name the cluster shows under in the SigNoz
Kubernetes views. Every `spec.collector.spec.env` key becomes an entry in the
collector's `env`.

## Forge

```bash
foundryctl forge -f agent/casting.yaml -p agent/pours
foundryctl forge -f deployment/casting.yaml -p deployment/pours
```

The pours land in `<kind>/pours/collectionagent/collector/<kind>/`, each
holding the chart's `values.yaml`. The collector config Foundry molds lives
inside it under `collectors.<kind>.config`.

## Cast

```bash
foundryctl cast -f agent/casting.yaml -p agent/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

This installs the chart with the Helm SDK using your current kubeconfig
context, creating the namespace on the first run. It detects an existing
release and upgrades instead of installing, so re-running is safe.

To inspect what would be installed first:

```bash
helm template signoz-collector-agent opentelemetry-kube-stack \
  --repo https://open-telemetry.github.io/opentelemetry-helm-charts --version 0.13.0 \
  -n signoz -f agent/pours/collectionagent/collector/agent/values.yaml
```

## Validate

```bash
kubectl get opentelemetrycollectors,daemonsets,deployments -n signoz
```

The operator creates the `signoz-collector-agent-collector` DaemonSet and the
`signoz-collector-deployment-collector` Deployment.

## Customization

Change collector settings through `spec.collector.spec.config.data`. For any
other chart value, such as resource limits or node selectors, use
[patches](../../../../concepts/patches.md) targeting `values.yaml`.

## Annotations

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-collector-controller` | `default` | Controller that owns the collector workload: `default` pours the DaemonSet or Deployment for the built-in Kubernetes controllers, `opentelemetry-operator` pours an `OpenTelemetryCollector` resource for the operator to reconcile |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the collector is deployed into |
| `foundry.signoz.io/kubernetes-helm-chart` | `opentelemetry-kube-stack` | Chart name in the repository, a URL to a chart archive, or a local chart path |
| `foundry.signoz.io/kubernetes-helm-repo-url` | `https://open-telemetry.github.io/opentelemetry-helm-charts` | Chart repository the chart name is resolved against; unused when the chart states its own location |
| `foundry.signoz.io/kubernetes-helm-chart-version` | `0.13.0` | Chart version to install; `latest` installs the newest chart in the repository |

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
    flavor: helm
  collector:
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <cluster-name>
```
