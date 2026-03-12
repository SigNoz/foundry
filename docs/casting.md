## Casting

A _casting_ is one YAML file that describes a full SigNoz deployment. Foundry fills in defaults; you override what you need.

### How to write `casting.yaml`

You’ll build the file in this order:

1. **Name your deployment**: `apiVersion` and `metadata` (name, optional annotations).
2. **Where it runs**: Deployment target: Docker, systemd, or Render.
3. **What runs**: Moldings (SigNoz, ingester, ClickHouse, metastore). Add blocks when you want to change defaults.
4. **How it's configured**: Per-molding `spec`: images, env, scaling, config files.
5. **Run it**: Point Foundry at the file and generate artifacts.


#### 1. Name your deployment

Top of the file: `apiVersion` and `metadata`.

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz-prod    # deployment ID: used as prefix for service names
  annotations: {}     # optional; required for systemd (step 2)
```

`name` is required: use something that identifies this deployment (`signoz-prod`, `signoz-dev`, whatever). `annotations` is optional unless you're on systemd/binary; then you'll put binary paths there (see step 2).

> [!TIP] 
> Short, environment-specific names work best; they end up in generated service names.

#### 2. Where it runs (deployment target)

`spec.deployment` tells Foundry how you're deploying: Docker Compose, systemd, or Render. It uses this to pick the right mold and spit out the right artifacts.

```yaml
spec:
  deployment:
    mode: docker      # or systemd
    flavor: compose   # or binary | blueprint
    platform:         # optional; "render" for Render
```

Pick one row:

| Where you're deploying | `mode`     | `flavor`    | `platform` |
| ---------------------- | ---------- | ----------- | --------- |
| **Docker Compose**      | `docker`   | `compose`   | (none)    |
| **Linux (systemd)**    | `systemd`  | `binary`    | (none)    |
| **Kubernetes (Kustomize)** | `kubernetes` | `kustomize` | (none)  |
| **[Render](https://render.com)** | (none) | `blueprint` | `render`  |

> [!NOTE] 
> systemd (`mode` + `flavor: binary`):** Foundry needs the paths to your binaries. Put them in `metadata.annotations`:
>
> | Annotation | What it's for |
> | ---------- | ------------- |
> | `foundry.signoz.io/signoz-binary-path` | SigNoz binary (for example, `/opt/signoz/bin/signoz`) |
> | `foundry.signoz.io/ingester-binary-path` | OTel Collector / ingester (for example, `/opt/ingester/bin/signoz-otel-collector`) |
> | `foundry.signoz.io/metastore-postgres-binary-path` | PostgreSQL binary when using Postgres metastore (for example, `/usr/bin/postgres`) |
>
> Example:
> 
> ```yaml
> metadata:
> name: signoz
> annotations:
>  foundry.signoz.io/signoz-binary-path: /opt/signoz/bin/signoz
>  foundry.signoz.io/ingester-binary-path: /opt/ingester/bin/signoz-otel-collector
>  foundry.signoz.io/metastore-postgres-binary-path: /usr/bin/postgres
> ```

#### 3. What runs (moldings)

_Moldings_ are the pieces (SigNoz, ingester, ClickHouse, etc.). Foundry has defaults for all of them; add a block under `spec` when you want to change one.

| Molding key in `spec` | Component |
| --------------------- | --------- |
| `signoz`              | SigNoz |
| `ingester`            | OTel Collector (ingestion & processing) |
| `telemetrystore`      | ClickHouse (logs, traces, metrics) |
| `telemetrykeeper`     | ClickHouse Keeper (coordination) |
| `metastore`           | Metadata store (PostgreSQL or SQLite) |

Angle brackets are placeholders: swap `<deployment-name>` for your ID, and pick valid `mode` / `flavor` / `platform` from the table above.

```yaml
apiVersion: v1alpha1
metadata:
  name: <deployment-name>
  annotations: {}   # optional; required for systemd with binary paths
spec:
  deployment:
    mode: <docker|systemd>
    flavor: <compose|binary|blueprint>
    platform: <render>   # optional
  # Override only what you need:
  signoz:
    spec: { ... }
  ingester:
    spec: { ... }
  telemetrystore:
    spec: { ... }
  telemetrykeeper:
    spec: { ... }
  metastore:
    kind: postgres   # or sqlite
    spec: { ... }
```

#### 4. How it’s configured (molding spec)

Override a molding by giving it a `spec` block. Whatever you set gets merged with Foundry's defaults.

**Fields you'll see:**

| Field               | Meaning |
| ------------------- | ------- |
| `enabled`           | Turn the component on/off (default: `true`) |
| `image`             | Container image (Docker mode) |
| `version`           | Version label (for example, for systemd or tagging) |
| `cluster.replicas`  | Number of replicas |
| `cluster.shards`    | Shards (TelemetryStore only) |
| `env`               | Environment variables (key/value map) |
| `config.data`       | Config files: **filename → file contents** |
| `config.knobs`      | Platform-specific knobs (see below) |

#### Knobs (platform-specific overrides)

Knobs let you tune platform-specific settings per component without writing raw patches. They live under `<component>.spec.config.knobs`.

Each casting defines which knobs it supports. For example, the **kustomize** casting supports:

| Knob | Type | Description |
|---|---|---|
| `resources` | `map` | CPU/memory requests and limits |
| `tolerations` | `list` | Pod tolerations |
| `nodeSelector` | `map` | Node selection constraints |
| `affinity` | `map` | Pod affinity/anti-affinity rules |
| `topologySpreadConstraints` | `list` | Topology spread constraints |
| `podSecurityContext` | `map` | Pod-level security context |
| `securityContext` | `map` | Container-level security context |
| `imagePullSecrets` | `list` | Image pull secrets |
| `minReadySeconds` | `int` | Min seconds before pod is considered ready |
| `storageSize` | `string` | PVC storage size (e.g. `"10Gi"`) |
| `storageClass` | `string` | PVC storage class name |
| `serviceType` | `string` | Kubernetes service type (e.g. `"LoadBalancer"`) |
| `serviceAnnotations` | `map` | Service annotations |
| `serviceLabels` | `map` | Service labels |
| `podAnnotations` | `map` | Pod annotations |
| `podLabels` | `map` | Pod labels |

Knobs are validated at forge time — unknown keys and type mismatches are caught before templates run.

Example:

```yaml
spec:
  signoz:
    spec:
      config:
        knobs:
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "1"
              memory: "1Gi"
          nodeSelector:
            node-role: observability
          serviceType: LoadBalancer
```

#### Patches (escape hatch)

Patches provide a casting-wide escape hatch for overriding any generated resource. They live at the top level of the casting (not per-component), and the casting's platform engine handles application.

For the **kustomize** casting, patches are rendered into the root `kustomization.yaml` and processed natively by kustomize. Patch files are referenced by path relative to the project root; foundry resolves them to the correct relative path for kustomize at forge time.

```yaml
patches:
  - path: patches/signoz-resources.yaml
```

Where `patches/signoz-resources.yaml` contains a standard strategic merge patch or JSON patch:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: signoz-prod-signoz
spec:
  template:
    spec:
      containers:
        - name: signoz
          resources:
            limits:
              memory: "4Gi"
```

> [!NOTE]
> Patches are an escape hatch for cases that knobs don't cover. Because patches target specific resource kinds and names, they require knowledge of the generated output structure. Prefer knobs for common overrides.

#### 5. Run it

When the file's done:

1. Run:

   ```shell
   foundry cast -f casting.yaml
   ```

2. Foundry merges your overrides with defaults and writes out the artifacts (Compose files, systemd units, or Render blueprint, depending on what you picked).

That's it. The casting file is the source of truth; Foundry does the rest.

## Examples

**Minimal: Docker Compose, all defaults:**

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
```

**Same, with a few overrides (images, scaling, env):**

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  signoz:
    spec:
      image: signoz/signoz:v0.110.0
  telemetrystore:
    spec:
      image: clickhouse/clickhouse-server:25.5.6
      cluster:
        replicas: 1
        shards: 1
```

**Kustomize with knobs and patches:**

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz-prod
spec:
  deployment:
    mode: kubernetes
    flavor: kustomize
  signoz:
    spec:
      config:
        knobs:
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
          serviceType: LoadBalancer
          storageSize: "50Gi"
          storageClass: gp3
  metastore:
    kind: postgres
    spec:
      config:
        knobs:
          storageSize: "20Gi"
          nodeSelector:
            node-role: database
patches:
  - path: patches/ingester-hpa.yaml
```
