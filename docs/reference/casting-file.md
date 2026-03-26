# Casting File Reference

Complete reference for the `casting.yaml` configuration file. For conceptual overview, see [Casting](../concepts/casting.md).

## Top-level structure

```yaml
apiVersion: v1alpha1
metadata:
  name: <string>              # required
  annotations: <map>          # optional
spec:
  deployment: <deployment>    # required
  signoz: <molding>
  ingester: <molding>
  telemetrystore: <molding>
  telemetrykeeper: <molding>
  metastore: <metastore>
  patches: <list>
```

## Deployment

Defines where and how SigNoz is deployed.

```yaml
spec:
  deployment:
    mode: <string>
    flavor: <string>
    platform: <string>
```

### Supported combinations

Each row is a valid combination. Mixing values across rows is not supported.

| Target | `mode` | `flavor` | `platform` |
| --- | --- | --- | --- |
| Docker Compose | `docker` | `compose` | - |
| Docker Swarm | `docker` | `swarm` | - |
| Systemd (binary) | `systemd` | `binary` | - |
| Kubernetes (Kustomize) | `kubernetes` | `kustomize` | - |
| Kubernetes (Helm) | `kubernetes` | `helm` | - |
| Render | - | `blueprint` | `render` |
| Coolify | - | `stack` | `coolify` |
| Railway | - | `template` | `railway` |
| AWS ECS (EC2) | `ec2` | `terraform` | `ecs` |

## Molding spec

Each molding (`signoz`, `ingester`, `telemetrystore`, `telemetrykeeper`) accepts a `spec` block:

```yaml
<molding>:
  spec:
    enabled: <bool>
    image: <string>
    version: <string>
    cluster:
      replicas: <int>
      shards: <int>
    env: <map>
    config:
      data: <map>
```

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | `true` | Include this component in the deployment |
| `image` | string | (built-in) | Container image (Docker and Kubernetes modes) |
| `version` | string | (built-in) | Version label (systemd mode, tagging) |
| `cluster.replicas` | int | `1` | Number of replicas |
| `cluster.shards` | int | `1` | Number of shards (TelemetryStore only) |
| `env` | map | `{}` | Environment variables as key-value pairs |
| `config.data` | map | `{}` | Config file overrides: filename to file contents |

## MetaStore

The metastore has an additional `kind` field to select the backend.

```yaml
spec:
  metastore:
    kind: <string>
    spec: <molding-spec>
```

| Kind | Backend | Notes |
| --- | --- | --- |
| `postgres` | PostgreSQL | Default. Recommended for production. |
| `sqlite` | SQLite | Embedded, single-node only. |

## Patches

List of patch operations applied to generated output files.

```yaml
spec:
  patches:
    - type: <string>
      target: <string>
      operations:
        - op: <string>
          path: <string>
          value: <any>
          from: <string>
```

| Field | Required | Description |
| --- | --- | --- |
| `type` | No | Patch driver. Default: `jsonpatch`. |
| `target` | Yes | Output file to patch. Exact path, basename, or glob. |
| `operations` | Yes | List of JSON Patch (RFC 6902) operations. |

See [Patches](../concepts/patches.md) for operation details and examples.

## Annotations

### Systemd binary paths

Required when using `mode: systemd`, `flavor: binary`.

| Annotation | Description |
| --- | --- |
| `foundry.signoz.io/signoz-binary-path` | Path to the SigNoz binary |
| `foundry.signoz.io/ingester-binary-path` | Path to the OTel Collector binary |
| `foundry.signoz.io/metastore-postgres-binary-path` | Path to the PostgreSQL binary |

```yaml
metadata:
  name: signoz
  annotations:
    foundry.signoz.io/signoz-binary-path: /opt/signoz/bin/signoz
    foundry.signoz.io/ingester-binary-path: /opt/ingester/bin/signoz-otel-collector
    foundry.signoz.io/metastore-postgres-binary-path: /usr/bin/postgres
```

### Kubernetes Helm annotations

Optional. Override the default Helm chart source when using `mode: kubernetes`, `flavor: helm`.

| Annotation | Default | Description |
| --- | --- | --- |
| `foundry.signoz.io/kubernetes-helm-casting-chart` | `signoz/signoz` | Helm chart reference |
| `foundry.signoz.io/kubernetes-helm-casting-repo-url` | `https://charts.signoz.io` | Helm chart repository URL |
| `foundry.signoz.io/kubernetes-helm-casting-repo-name` | `signoz` | Helm chart repository name |
| `foundry.signoz.io/kubernetes-helm-casting-forge-chart` | - | Set to `true` to download and bundle the chart locally during forge |

### ECS annotations

Required when using `platform: ecs`, `mode: ec2`, `flavor: terraform`.

| Annotation | Maps to tfvar | Description |
| --- | --- | --- |
| `foundry.signoz.io/ecs/region` | `region` | AWS region |
| `foundry.signoz.io/ecs/cluster-id` | `ecs_cluster_id` | ECS cluster ARN or ID |
| `foundry.signoz.io/ecs/subnet-ids` | `subnet_ids` | Comma-separated subnet IDs |
| `foundry.signoz.io/ecs/security-group-ids` | `security_group_ids` | Comma-separated security group IDs |
| `foundry.signoz.io/ecs/vpc-id` | `vpc_id` | VPC ID for Cloud Map namespace |
| `foundry.signoz.io/ecs/config-bucket` | `config_bucket` | S3 bucket for component configs |
| `foundry.signoz.io/ecs/task-role-arn` | `task_role_arn` | IAM role ARN for ECS tasks |
| `foundry.signoz.io/ecs/task-execution-role-arn` | `task_execution_role_arn` | IAM role ARN for task execution |
| `foundry.signoz.io/ecs/capacity-provider` | `capacity_provider` | ECS capacity provider name |

## Schema

The full JSON Schema for `casting.yaml` is available at [`docs/schemas/v1alpha1.yaml`](../schemas/v1alpha1.yaml).
