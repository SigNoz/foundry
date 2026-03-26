# Kustomize with Patches

| Field | Value |
| --- | --- |
| **Mode** | `kubernetes` |
| **Flavor** | `kustomize` |
| **Platform** | `-` |

## Overview

Extends the base [kubernetes/kustomize](../kustomize/) example with production patches for AWS. Demonstrates setting storage classes, resource limits, dedicated node scheduling, and a LoadBalancer service using `spec.patches`.

For patch reference, see [Patches](../../concepts/patches.md).

## Prerequisites

- Kubernetes cluster (1.24+)
- `kubectl` with kustomize support
- Cloud provider storage class (for example, `gp3` on AWS)

## Configuration

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    flavor: kustomize
    mode: kubernetes
  patches:
    # Storage class and size on ClickHouse data volumes
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

    # Resource limits on ClickHouse
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

    # Resource limits on SigNoz
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

    # Tolerations and nodeSelector for dedicated nodes
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

    # LoadBalancer with AWS NLB
    - target: "deployment/signoz/service.yaml"
      operations:
        - op: replace
          path: /spec/type
          value: LoadBalancer
        - op: add
          path: /metadata/annotations
          value:
            service.beta.kubernetes.io/aws-load-balancer-type: nlb

    # Resource limits on ingester
    - target: "deployment/ingester/deployment.yaml"
      operations:
        - op: replace
          path: /spec/template/spec/containers/0/resources
          value:
            requests:
              cpu: "1"
              memory: "2Gi"
            limits:
              cpu: "2"
              memory: "4Gi"

    # Storage class on PostgreSQL metastore
    - target: "deployment/metastore/postgres/statefulset.yaml"
      operations:
        - op: add
          path: /spec/volumeClaimTemplates/0/spec/storageClassName
          value: gp3
        - op: replace
          path: /spec/volumeClaimTemplates/0/spec/resources/requests/storage
          value: 20Gi
```

## Deploy

```bash
foundryctl cast -f casting.yaml
```

Or step by step:

```bash
foundryctl forge -f casting.yaml
kubectl apply -k pours/deployment/
```

## Generated output

Same structure as [kubernetes/kustomize](../kustomize/), with patches applied to the generated manifests.

## Customization

### Storage class and size

Sets the storage class on volume claim templates. Adjust for your cloud provider: `gp3` on AWS, `pd-ssd` on GCP, `managed-premium` on Azure.

ClickHouse has two volume claim templates (`data-0` and `default`). Both should use the same storage class.

### Resource limits

Sets CPU and memory requests/limits per component. The JSON Patch path varies by resource kind:

- ClickHouseInstallation: `/spec/templates/podTemplates/0/spec/containers/0/resources`
- StatefulSet or Deployment: `/spec/template/spec/containers/0/resources`

### Tolerations and nodeSelector

Schedules SigNoz on nodes labeled `node-role.kubernetes.io/observability` with a `dedicated=signoz:NoSchedule` toleration.

### LoadBalancer with AWS NLB

Changes the SigNoz Service to `LoadBalancer` with `service.beta.kubernetes.io/aws-load-balancer-type: nlb`.
