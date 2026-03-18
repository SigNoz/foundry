# Helm with Patches

| | |
|---|---|
| Deployment Mode | `kubernetes` / `helm` |
| Use Case | Kubernetes Helm deployment with custom resources, scheduling, and persistence |

## Overview

Customizes a Kubernetes SigNoz Helm deployment for AWS using `spec.patches`. Sets resource limits, dedicated node scheduling, and persistence configuration on the generated `values.yaml`.

For patch configuration reference, see [casting.md](../../../casting.md#patches-platform-specific-overrides).

## Configuration

This example patches the following Helm values:

| Patch | Description |
|---|---|
| [Resource limits](#resource-limits) | CPU and memory requests/limits for ClickHouse, SigNoz, and OTel Collector |
| [Tolerations and nodeSelector](#tolerations-and-nodeselector) | Schedule SigNoz on dedicated observability nodes |
| [Persistence](#persistence) | ClickHouse storage class and size |

### Resource limits

Sets resource requests and limits on components. Each component maps to a top-level key in the Helm chart values.

```yaml
patches:
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
```

Same pattern applies to `signoz` and `otelCollector`. See `casting.yaml` for all targets.

### Tolerations and nodeSelector

Schedules SigNoz on nodes labeled `node-role.kubernetes.io/observability` with a `dedicated=signoz:NoSchedule` toleration.

```yaml
patches:
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
```

### Persistence

Configures ClickHouse persistent storage with a cloud provider storage class.

```yaml
patches:
  - target: "deployment/values.yaml"
    operations:
      - op: add
        path: /clickhouse/persistence
        value:
          enabled: true
          storageClass: gp3
          size: 100Gi
```
