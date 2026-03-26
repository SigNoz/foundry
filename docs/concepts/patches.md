# Patches

Patches let you customize any generated output file using JSON Patch (RFC 6902) operations. They are applied during forge, after Foundry generates materials and before writing to `pours/`.

## Two-tier model

Foundry separates configuration into two layers:

- **`spec.<Molding>`** is the application domain. Foundry understands, validates, and enriches it.
- **`spec.patches`** is the platform domain. Foundry applies it as-is, no validation.

This gives you full coverage over the generated output without Foundry needing to model every platform-specific field. Use `spec.<Molding>` for component configuration (images, replicas, env, config files). Use `patches` for everything else (resource limits, storage classes, service types, scheduling constraints).

## Patch entry fields

| Field | Required | Description |
|---|---|---|
| `type` | No | Patch driver. Defaults to `jsonpatch`. |
| `target` | Yes | Output file to patch. Supports exact path, basename, or glob. |
| `operations` | Yes | List of JSON Patch (RFC 6902) operations. |

## JSON Patch operations

| Operation | Description | Required fields |
|---|---|---|
| `add` | Add a value at path. Append to array with `/-`. | `op`, `path`, `value` |
| `remove` | Remove the value at path. | `op`, `path` |
| `replace` | Replace the value at path. Path must exist. | `op`, `path`, `value` |
| `move` | Move a value from one path to another. | `op`, `from`, `path` |
| `copy` | Copy a value from one path to another. | `op`, `from`, `path` |
| `test` | Assert a value equals the given value. Fails if not. | `op`, `path`, `value` |

## Target matching

- **Exact path:** `target: "deployment/compose.yaml"`
- **Glob:** `target: "deployment/telemetrystore-*.yaml"` matches multiple files

> [!TIP]
> Run `foundryctl forge` first without patches to see the generated file names and structure, then write patches against them.

## Examples

### Docker Compose

Set a memory limit on ClickHouse and add a custom environment variable to SigNoz:

```yaml
spec:
  patches:
    - target: "compose.yaml"
      operations:
        - op: replace
          path: /services/clickhouse/mem_limit
          value: "4G"
        - op: add
          path: /services/signoz/environment/-
          value: "CUSTOM_VAR=value"
```

### Systemd

Change the restart policy and add a memory limit on a service unit:

```yaml
spec:
  patches:
    - target: "signoz-ingester.service"
      operations:
        - op: replace
          path: /Service/Restart
          value: always
        - op: add
          path: /Service/MemoryMax
          value: "4G"
```

### Kubernetes (Kustomize)

Set resource limits on ClickHouse:

```yaml
spec:
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

For complete Kubernetes examples with storage classes, tolerations, nodeSelector, and LoadBalancer configuration, see:
- [Kustomize with patches](../examples/kubernetes/kustomize-patches/)
- [Helm with patches](../examples/kubernetes/helm-patches/)

## When to use patches vs config.data

| Use case | Use |
|---|---|
| Application config files (OTel Collector config, ClickHouse config) | `config.data` in the molding spec |
| Platform files (compose files, service units, Kubernetes manifests, Helm values) | `spec.patches` |

> [!NOTE]
> Config files like `otel-collector-config.yaml` or `clickhouse-config.yaml` don't need patches. Use `config.data` in the [molding spec](moldings.md#custom-config-files) instead.

## Patches vs annotations

Patches modify generated output *after* generation. [Annotations](annotations.md) provide parameters *before* generation. If Foundry needs a value to generate files correctly (binary paths, AWS resource IDs), use annotations. If you want to tweak something in the generated output (resource limits, service types), use patches.

## Next steps

- [Annotations](annotations.md) - deployment-specific parameters (pre-generation)
- [Casting](casting.md) - the full casting file structure
- [Moldings](moldings.md) - component configuration
- [Casting file reference](../reference/casting-file.md) - complete field reference
