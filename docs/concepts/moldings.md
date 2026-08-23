# Moldings

Moldings are the individual components that make up a SigNoz deployment. Each molding maps to a service (ClickHouse, PostgreSQL, OTel Collector, etc.) and generates the configuration and deployment files for that service.

Foundry ships with sensible defaults for every molding. Override only what you need by adding a block under `spec` in your casting.

## Components

| Key in `spec` | Component | Role |
|---|---|---|
| `telemetrykeeper` | ClickHouse Keeper or ZooKeeper | Cluster coordination for ClickHouse |
| `telemetrystore` | ClickHouse | Stores logs, traces, and metrics |
| `metastore` | PostgreSQL or SQLite | Stores metadata (dashboards, alerts, users) |
| `signoz` | SigNoz | UI and API server |
| `ingester` | SigNoz OTel Collector | Receives, processes, and writes telemetry data |

### Processing order

Foundry processes moldings in dependency order during forge:

```
TelemetryKeeper -> TelemetryStore -> MetaStore -> SigNoz -> Ingester
```

This ensures each component's configuration can reference its dependencies.

## Configuring a molding

Override a molding by giving it a `spec` block. Whatever you set gets merged with Foundry's defaults.

```yaml
spec:
  telemetrystore:
    spec:
      image: clickhouse/clickhouse-server:25.12.5
      cluster:
        replicas: 1
        shards: 1
```

### Spec fields

| Field | Description |
|---|---|
| `enabled` | Turn the component on or off (default: `true`) |
| `image` | Container image (Docker and Kubernetes modes) |
| `version` | Version label (systemd mode, tagging) |
| `cluster.replicas` | Number of replicas |
| `cluster.shards` | Number of shards (TelemetryStore only) |
| `env` | Environment variables as a key-value map |
| `config.data` | Config file overrides merged with the generated config: filename to contents |

### Disabling a molding

Set `enabled: false` to exclude a component from the deployment:

```yaml
spec:
  telemetrykeeper:
    spec:
      enabled: false
```

### Custom config files

Use `config.data` to override a component's config files. The key is the name of a file Foundry generates; the value is a YAML fragment with only the changes. Foundry merges the fragment onto the generated file per [JSON Merge Patch (RFC 7386)](https://datatracker.ietf.org/doc/html/rfc7386):

- Scalars and maps deep-merge: your value wins, generated siblings survive.
- Setting a key to `null` deletes it.
- Lists are replaced wholesale: restate every member you want to keep.

Run `foundryctl forge` and read the generated file under `pours/` to see what a fragment merges onto.

Overriding a single value in the ingester's config:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  ingester:
    spec:
      config:
        data:
          ingester.yaml: |
            receivers:
              otlp:
                protocols:
                  grpc:
                    endpoint: 0.0.0.0:4317
```

Adding a receiver to a pipeline. `receivers` is a list, so the generated `otlp` member is restated:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  ingester:
    spec:
      config:
        data:
          ingester.yaml: |
            receivers:
              prometheus:
                config:
                  scrape_configs:
                    - job_name: my-app
                      static_configs:
                        - targets: ["localhost:8080"]
            service:
              pipelines:
                metrics:
                  receivers: [otlp, prometheus]
```

> [!WARNING]
> Writing `receivers: [prometheus]` above would replace the list and drop `otlp` from the pipeline.

To remove something, set its definition to `null` and restate the list that references it. Disabling the ingester's `pprof` extension:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
  ingester:
    spec:
      config:
        data:
          ingester.yaml: |
            extensions:
              pprof: null
            service:
              extensions: [signoz_health_check]
```

> [!NOTE]
> `config.data` overrides application-level config files that Foundry understands and manages. For platform-level files (compose files, service units, Kubernetes manifests), use [patches](patches.md) instead.

### TelemetryKeeper kind

The telemetry keeper supports two backends. Set the `kind` field to choose; the image and version default to the selected kind:

```yaml
spec:
  telemetrykeeper:
    kind: clickhousekeeper   # default
```

| Kind | Backend | Notes |
|---|---|---|
| `clickhousekeeper` | ClickHouse Keeper | Default. Lightweight, no JVM required. |
| `zookeeper` | ZooKeeper | Apache ZooKeeper, the widely operated JVM based coordination service. |

The `zookeeper` kind is not supported with `mode: systemd` yet.

### MetaStore kind

The metastore supports two backends. Set the `kind` field to choose:

```yaml
spec:
  metastore:
    kind: postgres   # default
    spec:
      image: postgres:16
```

| Kind | Backend | Notes |
|---|---|---|
| `postgres` | PostgreSQL | Default. Recommended for production. |
| `sqlite` | SQLite | Embedded, no external dependency. Single-node only. |

## Next steps

- [Casting](casting.md) - the full casting file structure
- [Annotations](annotations.md) - deployment-specific parameters
- [Patches](patches.md) - platform-level overrides on generated files
- [MCP server](mcp-server.md) - deploy the SigNoz MCP server alongside the stack
- [Casting file reference](../reference/casting-file.md) - complete field reference
