# Helm with Patches

| Field | Value |
| --- | --- |
| **Mode** | `kubernetes` |
| **Flavor** | `helm` |
| **Platform** | `-` |

## Overview

Extends the base [kubernetes/helm](../helm/) example with production patches for AWS. Demonstrates setting resource limits, dedicated node scheduling, and persistence configuration on the generated `values.yaml` using `spec.patches`.

For patch reference, see [Patches](../../concepts/patches.md).

## Prerequisites

- Kubernetes cluster (1.24+)
- Helm 3.x
- SigNoz Helm chart repo (`https://charts.signoz.io`)

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: helm
    mode: kubernetes
  patches:
    # Resource limits on ClickHouse
    - target: "deployment/values.yaml"
      operations:
        - op: add
          path: /clickhouse/resources
          value:
            requests:
              cpu: "2"
              memory: "4Gi"
            limits:
              cpu: "4"
              memory: "8Gi"

    # Resource limits on SigNoz
    - target: "deployment/values.yaml"
      operations:
        - op: add
          path: /signoz/resources
          value:
            requests:
              cpu: "500m"
              memory: "1Gi"
            limits:
              cpu: "1"
              memory: "2Gi"

    # Tolerations and nodeSelector for dedicated nodes
    - target: "deployment/values.yaml"
      operations:
        - op: add
          path: /signoz/tolerations
          value:
            - key: "dedicated"
              operator: "Equal"
              value: "signoz"
              effect: "NoSchedule"
        - op: add
          path: /signoz/nodeSelector
          value:
            node-role.kubernetes.io/observability: ""

    # Resource limits on OTel Collector (ingester)
    - target: "deployment/values.yaml"
      operations:
        - op: add
          path: /otelCollector/resources
          value:
            requests:
              cpu: "1"
              memory: "2Gi"
            limits:
              cpu: "2"
              memory: "4Gi"

    # ClickHouse persistence
    - target: "deployment/values.yaml"
      operations:
        - op: add
          path: /clickhouse/persistence
          value:
            enabled: true
            storageClass: gp3
            size: 100Gi
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
foundryctl forge -f casting.yaml
helm install signoz signoz/signoz -f pours/deployment/values.yaml -n signoz --create-namespace
```

## Generated output

```text
pours/deployment/
  values.yaml
```

## Customization

### Resource limits

Each component maps to a top-level key in the Helm chart values: `clickhouse`, `signoz`, `otelCollector`. Set `resources.requests` and `resources.limits` on each.

### Tolerations and nodeSelector

Schedules SigNoz on nodes labeled `node-role.kubernetes.io/observability` with a `dedicated=signoz:NoSchedule` toleration.

### Persistence

Configures ClickHouse persistent storage with a cloud provider storage class. Adjust `storageClass` for your provider: `gp3` on AWS, `pd-ssd` on GCP, `managed-premium` on Azure.
