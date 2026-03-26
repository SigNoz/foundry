# Casting

A casting is one YAML file that describes a complete SigNoz deployment. Foundry reads it, merges your overrides with built-in defaults, and generates everything needed to run the stack.

The casting file is the single source of truth for your deployment. You define what you want; Foundry handles the rest.

## Structure

A casting has five parts:

1. **Metadata** - name your deployment and set annotations
2. **Deployment target** - where it runs (Docker, Kubernetes, systemd, cloud platform)
3. **Moldings** - which components to include and how to configure them
4. **Annotations** - deployment-specific parameters Foundry reads during generation
5. **Patches** - platform-level overrides on the generated output

```yaml
apiVersion: v1alpha1
metadata:
  name: <deployment-name>
spec:
  deployment:
    mode: <mode>
    flavor: <flavor>
    platform: <platform>       # only for cloud platforms
  signoz:
    spec: { ... }
  ingester:
    spec: { ... }
  telemetrystore:
    spec: { ... }
  telemetrykeeper:
    spec: { ... }
  metastore:
    kind: postgres             # or sqlite
    spec: { ... }
  patches: [ ... ]
```

Replace `<deployment-name>` with an identifier for this deployment (for example, `signoz-prod`). This name is used as a prefix in generated service names.

## Metadata

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz-prod
  annotations: {}              # optional; required for some deployment modes
```

`name` is required. Short, environment-specific names work best since they end up in generated service names.

`annotations` is optional for most deployment modes. Some modes (systemd, ECS, Helm) use annotations to pass deployment-specific parameters. See [Annotations](annotations.md) for when and how to use them.

## Deployment target

`spec.deployment` tells Foundry how you're deploying. It picks the right generator and produces the right artifacts.

```yaml
spec:
  deployment:
    mode: docker
    flavor: compose
```

Each row below is a valid combination. Mixing values across rows is not supported.

| Target | `mode` | `flavor` | `platform` |
|---|---|---|---|
| Docker Compose | `docker` | `compose` | - |
| Docker Swarm | `docker` | `swarm` | - |
| Systemd (binary) | `systemd` | `binary` | - |
| Kubernetes (Kustomize) | `kubernetes` | `kustomize` | - |
| Kubernetes (Helm) | `kubernetes` | `helm` | - |
| Render | - | `blueprint` | `render` |
| Coolify | - | `stack` | `coolify` |
| Railway | - | `template` | `railway` |
| AWS ECS (EC2) | `ec2` | `terraform` | `ecs` |

> [!TIP]
> Run `foundryctl gen examples` to generate a working `casting.yaml` for every supported deployment mode.

## Moldings

Moldings are the individual components of a SigNoz deployment. Foundry has defaults for all of them. Add a block under `spec` only when you want to change something.

See [Moldings](moldings.md) for details on each component and how to configure them.

## Annotations

Annotations provide deployment-specific parameters that Foundry reads during generation. They are inputs to the pipeline, not modifications to the output.

Most deployment modes don't require annotations. Systemd and ECS do. Helm supports optional chart overrides.

See [Annotations](annotations.md) for the full guide, including how annotations differ from patches.

## Patches

Patches let you customize any generated output file without Foundry needing to model every platform-specific field. They use JSON Patch (RFC 6902) operations.

See [Patches](patches.md) for the full guide.

## Minimal example

Docker Compose with all defaults:

```yaml
apiVersion: v1alpha1
metadata:
  name: signoz
spec:
  deployment:
    mode: docker
    flavor: compose
```

With overrides for images and scaling:

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

## Next steps

- [Moldings](moldings.md) - configure individual components
- [Annotations](annotations.md) - deployment-specific parameters
- [Patches](patches.md) - platform-level overrides
- [Casting file reference](../reference/casting-file.md) - complete field-by-field reference
- [Examples](../examples/) - working examples for every deployment mode
