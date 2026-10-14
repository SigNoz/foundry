# GKE Autopilot Collection Agent with Kustomize

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `gcp` |
| **Mode** | `kubernetes-auto` |
| **Flavor** | `kustomize` |

## Overview

Deploys the SigNoz Collection Agents on a GKE Autopilot cluster as plain
manifests composed by kustomize, the same way the
[`kubernetes/kustomize` casting](../../../kubernetes/kustomize/README.md) does,
with the agent shaped the way the SigNoz `k8s-infra` chart shapes it under
`global.cloud: gcp/autogke`. It adds the Google Cloud resource detectors to
both collectors:

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

Autopilot denies the `nodes/proxy` access that kubelet metadata labels and the
request and limit utilization metrics need, and admits a host path only under
`/var/log`, so the agent's receivers follow the chart's Autopilot settings:

- `kubeletstats` scrapes node, pod, container and volume metrics from the
  kubelet on its authenticated port 10250 with the agent's service account,
  with no `extra_metadata_labels`
- `kubeletstats` turns off the `k8s.pod.*` and `k8s.container.*`
  `cpu_request_utilization`, `cpu_limit_utilization`,
  `memory_request_utilization` and `memory_limit_utilization` metrics
- `hostmetrics` runs with no `root_path`, reading `/proc` and the mounts the
  agent's container sees
- Pod logs come from `/var/log/pods`, the agent's only host path

The agent DaemonSet binds no host ports, so applications send OTLP to the
agent's Service. With `metadata.name: signoz`, that is
`signoz-collector-agent.signoz.svc.cluster.local`, on port 4317 (gRPC) or 4318
(HTTP). The Service keeps traffic on the node it starts from, so each
application reaches the agent on its own node.

## Prerequisites

- A GKE Autopilot cluster and `kubectl` configured against it
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
    mode: kubernetes-auto
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
    platform: gcp
    mode: kubernetes-auto
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

The generated output and the annotations are those of the
[`kubernetes/kustomize` casting](../../../kubernetes/kustomize/README.md). With
`foundry.signoz.io/kubernetes-collector-controller: opentelemetry-operator`
stated, the collectors run as `OpenTelemetryCollector` resources that the
[OpenTelemetry Operator](../../../kubernetes/kustomize-opentelemetry-operator/README.md)
reconciles, with the same Autopilot shape (no host ports, and `/var/log` as
the agent's only host path) and the same collector config. The operator then
serves the agent through the `signoz-collector-agent-collector` Service it
creates, on ports 4317 (gRPC) and 4318 (HTTP), with traffic kept on the node
it starts from.
