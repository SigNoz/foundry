# EKS Collection Agent with Kustomize

| Field | Value |
| --- | --- |
| **Kind** | `CollectionAgent` |
| **Platform** | `aws` |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |

## Overview

Deploys the SigNoz Collection Agents on an EKS cluster as plain manifests
composed by kustomize, the same way the
[`kubernetes/kustomize` casting](../../../kubernetes/kustomize/README.md) does,
and adds the AWS resource detectors to both collectors:

- `agent/` runs the detectors `k8snode`, `env`, `eks`, `ec2`, `system`
- `deployment/` runs the detectors `env`, `eks`, `ec2`

Signals then carry `cloud.provider` (`aws`), `cloud.platform` (`aws_eks`),
`cloud.account.id`, and the location of the node the collector runs on as
`cloud.region` and `cloud.availability_zone`. The agent's signals also carry
the node's instance attributes `host.type` and `host.image.id`; the deployment
runs once per cluster and carries no host attributes.

`host.name` stays the node name and `k8s.cluster.name` stays
`K8S_CLUSTER_NAME`. A key stated in `OTEL_RESOURCE_ATTRIBUTES` wins over the
detected value.

> [!NOTE]
> Detection runs once, when a collector starts. The `ec2` detector reads the
> EC2 instance metadata service from the pod, and a collector that cannot reach
> it starts without the attributes it provides. On nodes that require IMDSv2,
> set the instance metadata hop limit to 2 so pods reach the service.

## Prerequisites

- An EKS cluster and `kubectl` configured against it
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
    platform: aws
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
    platform: aws
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
[`kubernetes/kustomize` casting](../../../kubernetes/kustomize/README.md), and
the `opentelemetry-operator` controller is that of the
[operator example](../../../kubernetes/kustomize-opentelemetry-operator/README.md).
