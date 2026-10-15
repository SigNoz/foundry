# GKE Collection Agent with Helm

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `gcp` |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |

## Overview

Deploys the SigNoz Collection Agents on a GKE cluster through the
SigNoz `k8s-infra` Helm chart, the same way the
[`kubernetes/helm` casting](../../../kubernetes/helm/README.md) does, and adds
the Google Cloud resource detectors to both collectors:

- `agent/` runs the detectors `k8snode`, `env`, `gcp`, `system`
- `deployment/` runs the detectors `env`, `gcp`

Signals then carry `cloud.provider` (`gcp`), `cloud.platform`
(`gcp_kubernetes_engine`), `cloud.account.id` (the project), and the location
as `cloud.region` or `cloud.availability_zone`. The deployment runs once per
cluster and carries no host attributes.

`host.name` stays the node name and `k8s.cluster.name` stays
`K8S_CLUSTER_NAME`. A key stated in `OTEL_RESOURCE_ATTRIBUTES` wins over the
detected value.

> [!NOTE]
> Detection runs once, when a collector starts, and reads the metadata server
> from the pod. A collector that cannot reach it starts without the cloud
> attributes.

## Prerequisites

- A GKE cluster and a kubeconfig context pointing at it
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
    platform: gcp
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
    platform: gcp
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
Kubernetes views.

## Forge

From this directory:

```bash
foundryctl forge -f agent/casting.yaml -p agent/pours
foundryctl forge -f deployment/casting.yaml -p deployment/pours
```

## Cast

From this directory:

```bash
foundryctl cast -f agent/casting.yaml -p agent/pours
foundryctl cast -f deployment/casting.yaml -p deployment/pours
```

The generated output, the Helm releases, the annotations and the
`opentelemetry-operator` controller are those of the
[`kubernetes/helm` casting](../../../kubernetes/helm/README.md).
