# Kubernetes Collection Agent with Helm

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |

## Overview

Deploys the SigNoz Collection Agents on a Kubernetes cluster through the SigNoz
`k8s-infra` Helm chart, exporting to any SigNoz: Self-Hosted Community,
Self-Hosted Enterprise, or SigNoz Cloud.

Foundry owns the collector config and the chart is the platform surface: every
chart preset is turned off and the collector config Foundry molds is passed to
the chart as `otelAgent.config` / `otelDeployment.config`. The rendered
ConfigMap is the config in `casting.yaml.lock`, key for key, so the two
Kubernetes flavors deploy the same collector.

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
  - OTLP intake on the service's ports 4317 (gRPC) and 4318 (HTTP)

Run both. The SigNoz Kubernetes views require both sources: entity resolution
breaks when either the kubelet metrics or the cluster-level metrics are
missing.

Each kind installs as its own Helm release, `<metadata.name>-collector-agent`
and `<metadata.name>-collector-deployment`, into the `<metadata.name>`
namespace, or into the namespace the
`foundry.signoz.io/kubernetes-namespace` annotation names. Both castings share
the `metadata.name`, so the two releases compose into one namespace without
upgrading over each other.

Kubernetes metadata lands on every signal through the `k8sattributes`
processor. The chart stamps a `checksum/config` pod annotation from the
ConfigMap, so a config change rolls the workload on the next `cast`.

## Prerequisites

- A Kubernetes cluster and a kubeconfig context pointing at it
- Helm 3.x
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
    flavor: helm
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
    flavor: helm
  collector:
    kind: deployment
    spec:
      env:
        SIGNOZ_INGESTION_ENDPOINT: http://<signoz-host>:4318
        K8S_CLUSTER_NAME: <cluster-name>
```

`K8S_CLUSTER_NAME` names the cluster on every signal as `k8s.cluster.name`.
Set it: the SigNoz Kubernetes views require it on every entity, and queries
fail silently without it. It becomes the chart's `clusterName`; every other
`spec.collector.spec.env` key becomes an entry in the component's
`additionalEnvs`.

Extra resource attributes, `deployment.environment` among them, travel on
`OTEL_RESOURCE_ATTRIBUTES` in `spec.collector.spec.env`. Host and cluster
identity is set by the collector config itself, so your value only adds
attributes:

```yaml
      env:
        OTEL_RESOURCE_ATTRIBUTES: deployment.environment=production
```

## Forge

```bash
foundryctl forge -f agent/casting.yaml -p agent/pours
foundryctl forge -f deployment/casting.yaml -p deployment/pours
```

Each collector kind pours one values file at
`<kind>/pours/collectionagent/collector/<kind>/values.yaml`. A casting file
declaring both kinds pours a values file per document, and casting installs
each as its own release.

## Generated output

```text
<kind>/pours/collectionagent/collector/<kind>/
  values.yaml
```

The collector config Foundry molds lives inside that file, under
`otelAgent.config` for the agent and `otelDeployment.config` for the
deployment.

Every chart preset is turned off, so collector customization goes through
`spec.collector.spec.config.data` in the casting and platform customization
through `spec.patches` targeting `values.yaml`. The `helm template` command
under Cast renders the ConfigMap the chart builds from these values.

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
helm template signoz-collector-agent k8s-infra --repo https://charts.signoz.io \
  -n signoz -f agent/pours/collectionagent/collector/agent/values.yaml
```

## Annotations

Optional annotations naming the namespace and the chart source. These are not
required for standard deployments.

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-namespace` | `metadata.name` | Namespace the collector is deployed into |
| `foundry.signoz.io/kubernetes-helm-chart` | `k8s-infra` | Chart name in the repository, a URL to a chart archive, or a local chart path |
| `foundry.signoz.io/kubernetes-helm-repo-url` | `https://charts.signoz.io` | Chart repository the chart name is resolved against; unused when the chart states its own location |
| `foundry.signoz.io/kubernetes-helm-chart-version` | `latest` | Chart version to install; `latest` installs the newest chart in the repository |

Example pinning the chart version:

```yaml
apiVersion: v1alpha1
kind: CollectionAgent
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/kubernetes-helm-chart-version: 0.17.1
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
