# Kustomize with Patches

| | |
|---|---|
| Deployment Mode | `kubernetes` / `kustomize` |
| Use Case | Kubernetes deployment with custom storage, resources, scheduling, and networking |

## Overview

Customizes a Kubernetes SigNoz deployment for AWS using `spec.patches`. Sets storage classes for persistent volumes, resource limits, dedicated node scheduling, and a LoadBalancer service.

For patch configuration reference, see [casting.md](../../../casting.md#patches-platform-specific-overrides).

## Configuration

This example patches the following files:

| Patch | Target file |
|---|---|
| [Storage class and size](#storage-class-and-size) | `deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml`, `deployment/metastore/postgresql/statefulset.yaml` |
| [Resource limits](#resource-limits) | `deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml`, `deployment/signoz/statefulset.yaml`, `deployment/ingester/deployment.yaml` |
| [Tolerations and nodeSelector](#tolerations-and-nodeselector) | `deployment/signoz/statefulset.yaml` |
| [LoadBalancer with AWS NLB](#loadbalancer-with-aws-nlb) | `deployment/signoz/service.yaml` |

### Storage class and size

Sets storage class on volume claim templates. Required for cloud providers: `gp3` on AWS, `pd-ssd` on GCP, `managed-premium` on Azure.

ClickHouse has two volume claim templates (`data-0` and `default`):

```yaml
patches:
  - target: "deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml"
    operations:
      - op: add
        path: /spec/templates/volumeClaimTemplates/0/spec/storageClassName
        value: gp3
      - op: replace
        path: /spec/templates/volumeClaimTemplates/0/spec/resources/requests/storage
        value: 100Gi
      - op: add
        path: /spec/templates/volumeClaimTemplates/1/spec/storageClassName
        value: gp3
```

Same pattern applies to PostgreSQL metastore. See `casting.yaml` for all targets.

### Resource limits

Sets resource requests and limits on a container. The path varies by resource kind.

For a ClickHouseInstallation (pod template):

```yaml
patches:
  - target: "deployment/telemetrystore/clickhouse/clickhouseinstallation.yaml"
    operations:
      - op: replace
        path: /spec/templates/podTemplates/0/spec/containers/0/resources
        value:
          requests:
            cpu: "2"
            memory: "4Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
```

For a StatefulSet or Deployment (`/spec/template/spec/...`):

```yaml
patches:
  - target: "deployment/signoz/statefulset.yaml"
    operations:
      - op: replace
        path: /spec/template/spec/containers/0/resources
        value:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1"
```

Same pattern applies to `deployment/ingester/deployment.yaml`. See `casting.yaml` for all targets.

### Tolerations and nodeSelector

Schedules SigNoz on nodes labeled `node-role.kubernetes.io/observability` with a `dedicated=signoz:NoSchedule` toleration.

```yaml
patches:
  - target: "deployment/signoz/statefulset.yaml"
    operations:
      - op: add
        path: /spec/template/spec/tolerations
        value:
          - key: "dedicated"
            operator: "Equal"
            value: "signoz"
            effect: "NoSchedule"
      - op: add
        path: /spec/template/spec/nodeSelector
        value:
          node-role.kubernetes.io/observability: ""
```

### LoadBalancer with AWS NLB

Changes SigNoz Service to `LoadBalancer` with `service.beta.kubernetes.io/aws-load-balancer-type: nlb` annotation.

```yaml
patches:
  - target: "deployment/signoz/service.yaml"
    operations:
      - op: replace
        path: /spec/type
        value: LoadBalancer
      - op: add
        path: /metadata/annotations
        value:
          service.beta.kubernetes.io/aws-load-balancer-type: nlb
```
