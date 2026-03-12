# Casting Overrides Design Document

## Problem Statement

Foundry generates deployment files from a single `casting.yaml`. The current `MoldingSpec` models SigNoz component configuration (image, version, replicas, env, config). But platform engineers deploying to Kubernetes need to express platform-specific concerns — tolerations, resource requests/limits, storage classes, node selectors, service types — that don't exist in `MoldingSpec` and differ across deployment modes.

The question: how does foundry handle this growing surface area of platform-specific configuration without its schema gradually mirroring every platform's API?

## Core Insight

**Foundry's value is hiding SigNoz's internal wiring, not hiding the deployment platform.**

A platform engineer who needs tolerations already knows Kubernetes. They chose it deliberately. They're using foundry because figuring out how 5 SigNoz components interconnect (ClickHouse ↔ Keeper ↔ Ingester ↔ SigNoz ↔ MetaStore) is painful — not because they can't write a Deployment YAML.

The abstraction boundary:
- **`spec`** = SigNoz domain (foundry abstracts this for everyone)
- **Platform tuning** = platform domain (speak the platform's native language)

## Approaches Explored and Rejected

### Approach 1: Typed fields in MoldingSpec

Add `Resources`, `Tolerations`, `Storage` fields to `MoldingSpec`.

```go
type MoldingSpec struct {
    // ...existing...
    Resources    *ResourceRequirements
    Tolerations  []Toleration
    NodeSelector map[string]string
}
```

**Rejected because:**
- It never stops: resources today, then storage, then affinity, then probes, then service types, then PDBs, then ingress. The schema gradually mirrors the Kubernetes API.
- Even if framed as "universal," the motivation is K8s-specific. Docker compose resource limits and systemd cgroup controls don't map 1:1.
- Every new platform concept = Go type change + schema change + API version consideration.

### Approach 2: Annotations

Use `metadata.annotations` with naming conventions like `foundry.signoz.io/telemetrystore-storage-size: "100Gi"`.

**Rejected because:**
- Flat key-value (`map[string]string`) for structured, per-component data gets messy fast.
- Dozens of annotations for basic K8s needs (resources × components, storage × components).
- Annotations are global (on `metadata`), not per-component — requires naming conventions to scope them.
- Tolerations are YAML arrays — encoding them as annotation strings is YAML-in-string.

### Approach 3: `config.data` as the override mechanism

Each component's `config.data` gets platform-specific keys like `cpu-request: "500m"`.

**Rejected because:**
- `config.data` already has a job — it holds application configuration (OTel collector pipelines, ClickHouse config files) that moldings produce.
- Overloading it to also mean "platform deployment knobs" conflates two different concerns.
- `map[string]string` can't express structured data like tolerations without YAML-in-YAML.
- Convention-based key names with no validation or discoverability.

### Approach 4: YAML-in-YAML blob

`config.data["values.yaml"]` containing raw platform config as a YAML string.

**Rejected because:**
- YAML-in-YAML is ugly and error-prone.
- Users need to know the internal manifest/chart structure.
- No validation, no schema, no IDE support.

### Approach 5: Separate file alongside casting.yaml

Users place a `values.yaml` or `overlay.yaml` next to their `casting.yaml`.

**Rejected as primary mechanism because:**
- Breaks the single-file declarative model.
- Users still need platform knowledge to write the file.
- However, this is valid as an **escape hatch** for advanced customization (see "Patches" section — `path` references to external files).

### Approach 6: Foundry-invented abstraction language

Define foundry's own override vocabulary that gets translated per-casting:
```yaml
schedulingConstraints:
  - type: toleration
    key: dedicated
compute:
  cpuRequest: 500m
```

**Rejected because:**
- Creates a new language users must learn ON TOP of their platform.
- The same intent maps differently per platform in ways that matter (K8s has request vs limit distinction; docker just has limits; systemd has hard vs soft cgroups).
- Flattening these into one abstraction loses information that platform engineers specifically chose to control.

## Chosen Approach

### Three-part architecture: `spec` + platform field + `patches`

The casting.yaml has three kinds of configuration:

1. **`spec`** (per-component) — Foundry's domain. SigNoz topology, images, versions, replicas, env vars, application config. Understood by all castings. Flows through the foundry pipeline (enrich → mold → merge → forge).

2. **Platform field** (per-component, name TBD — see Naming section) — Casting's domain. Common knobs — flattened keys for the 90% case (`tolerations`, `resources`, `storageSize`, etc.). Named using the platform's native terminology. Foundry templates know where to place each key. Foundry core carries it as `map[string]any`; only the active casting's templates and validation read it.

3. **`patches`** (top-level, casting-wide) — Platform-native escape hatch for anything the common knobs don't cover. Supports both inline content and `path` references to external files. Patches use target selectors to specify which resources they apply to, so they don't need to be scoped per-component. The content format is casting-dependent — kustomize patches for kustomize casting, additional values files for helm, compose overrides for docker, drop-in files for systemd.

```yaml
signoz:
  spec: { ... }
  <name>:                    # per-component common knobs
    tolerations: [...]
    resources: { ... }

telemetrystore:
  spec: { ... }
  <name>:
    storageSize: 100Gi

patches:                     # top-level, casting-wide
- patch: |
    ...
  target:
    kind: StatefulSet
    name: signoz-signoz
- path: patches/production.yaml
```

### Why this works

- `spec` stays deployment-agnostic — no K8s types leak into docker/systemd castings.
- Common knobs use the platform's native field names — a K8s user writes `tolerations`, a docker user writes `memLimit`, a systemd user writes `memoryMax`.
- `patches` is top-level because patches use target selectors — a single patch can target any resource across components. This mirrors how kustomize's own `patches` field works at the kustomization level, not per-resource.
- `patches` speaks the platform's native patch language — its content format is casting-dependent, so the user writes what they already know.
- Everything needed for `Cast()` is in casting.yaml.
- Adding a new common knob = adding a template conditional (today) or a mapping entry (future). No Go type changes in `api/v1alpha1`.
- Advanced use cases don't require growing the common knob inventory — users drop to `patches`.

## How the platform tuning field works

### At the API level

```go
// api/v1alpha1 — foundry core carries it opaquely
type SigNoz struct {
    Spec   MoldingSpec    `json:"spec" yaml:"spec"`
    Status SigNozStatus   `json:"status" yaml:"status,omitempty"`
    // Name TBD — see Naming section
    <Name> map[string]any `json:"<name>,omitempty" yaml:"<name>,omitempty"`
}
```

### At the casting level

Each casting defines a typed struct for validation and schema generation:

```go
// internal/casting/kuberneteskustomizecasting/overrides.go
type KustomizeComponentOverrides struct {
    // Common knobs — platform-native field names for the 90% case
    Resources          map[string]any    `json:"resources,omitempty"`
    Tolerations        []map[string]any  `json:"tolerations,omitempty"`
    NodeSelector       map[string]string `json:"nodeSelector,omitempty"`
    StorageSize        string            `json:"storageSize,omitempty"`
    StorageClass       string            `json:"storageClass,omitempty"`
    ServiceType        string            `json:"serviceType,omitempty"`
    ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`
    PodAnnotations     map[string]string `json:"podAnnotations,omitempty"`

    // Escape hatch — platform-native patches for anything common knobs don't cover
    Patches            []Patch           `json:"patches,omitempty"`
}

// Patch supports both inline content and external file references.
// Modeled after kustomize's patches field (see kustomization.json schema).
type Patch struct {
    // Inline patch content (strategic merge patch or JSON patch)
    Patch  string       `json:"patch,omitempty"`
    // Path to an external patch file
    Path   string       `json:"path,omitempty"`
    // Optional target selector (kind, name, labelSelector, etc.)
    Target *PatchTarget `json:"target,omitempty"`
}

type PatchTarget struct {
    Group               string `json:"group,omitempty"`
    Version             string `json:"version,omitempty"`
    Kind                string `json:"kind,omitempty"`
    Name                string `json:"name,omitempty"`
    Namespace           string `json:"namespace,omitempty"`
    LabelSelector       string `json:"labelSelector,omitempty"`
    AnnotationSelector  string `json:"annotationSelector,omitempty"`
}
```

The casting unmarshals `map[string]any` into this struct during `Forge()` for validation. Unknown keys produce warnings/errors. The struct also drives `foundryctl schema` generation.

The `Patch` and `PatchTarget` types mirror kustomize's own schema (from [kustomization.json](https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/kustomization.json)). This means a user writing patches for foundry is writing the same YAML they'd write in a `kustomization.yaml` — no new vocabulary to learn.

### At the template level

**Common knobs** — templates read them with conditionals:

```yaml
{{- with $.Spec.Signoz.<Name> }}
  {{- if .tolerations }}
      tolerations:
{{ toYaml .tolerations | indent 8 }}
  {{- end }}
{{- end }}
```

**Patches** — foundry injects them into the generated `kustomization.yaml` for the component:

```yaml
# generated signoz/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- statefulset.yaml
- service.yaml
- serviceaccount.yaml
patches:
- target:
    kind: StatefulSet
    name: signoz-signoz
  patch: |
    apiVersion: apps/v1
    kind: StatefulSet
    metadata:
      name: signoz-signoz
    spec:
      template:
        spec:
          topologySpreadConstraints:
          - maxSkew: 1
            topologyKey: topology.kubernetes.io/zone
            whenUnsatisfiable: DoNotSchedule
```

This keeps base templates clean and uses kustomize's own patch engine for advanced customization. The common knobs are handled by templates; the patches are passed through to kustomize.

**Future**: common knobs could also be translated into kustomize patches (rather than template conditionals), making templates completely override-free. This is a non-breaking internal refactor — the casting.yaml schema stays identical.

## Vocabulary per casting

Each casting's field uses the platform's native terminology. The user doesn't learn a foundry-specific language — they write what they already know. Each casting also supports `patches` as the platform-native escape hatch.

### Kustomize casting (output = K8s manifests)

Common knobs use K8s API field names, flattened by concern (not full manifest paths):

| Key | Placed in | Default |
| --- | --- | --- |
| `resources` | `containers[0].resources` | per-component hardcoded |
| `tolerations` | `spec.template.spec.tolerations` | omitted |
| `nodeSelector` | `spec.template.spec.nodeSelector` | omitted |
| `storageSize` | `volumeClaimTemplates[].resources.requests.storage` | 10Gi/25Gi |
| `storageClass` | `volumeClaimTemplates[].storageClassName` | cluster default |
| `serviceType` | `service.spec.type` | ClusterIP |
| `serviceAnnotations` | `service.metadata.annotations` | omitted |
| `podAnnotations` | `template.metadata.annotations` | omitted |

Keys are flattened because full manifest paths (`spec.template.spec.tolerations`) are verbose and require knowing the exact resource structure. The template knows where to place each key.

`patches` for kustomize casting uses kustomize's own patch schema — inline strategic merge patches or JSON patches, with optional `target` selectors, or `path` references to external patch files. These are injected into the generated component `kustomization.yaml`.

```yaml
signoz:
  spec: { ... }
  <name>:
    # Common knobs
    tolerations:
    - key: dedicated
      value: signoz
      effect: NoSchedule
    resources:
      requests:
        cpu: "500m"
        memory: "512Mi"
    # Escape hatch — kustomize-native patches
    patches:
    - patch: |
        apiVersion: apps/v1
        kind: StatefulSet
        metadata:
          name: signoz-signoz
        spec:
          template:
            spec:
              topologySpreadConstraints:
              - maxSkew: 1
                topologyKey: topology.kubernetes.io/zone
                whenUnsatisfiable: DoNotSchedule
    - path: patches/signoz-pdb.yaml
      target:
        kind: StatefulSet
```

### Helm casting (output = values.yaml)

Helm charts already define a user-friendly knob layer (values.yaml). Common knobs ARE chart values — no foundry-invented vocabulary needed.

`patches` for helm casting points to additional values file(s) that get merged:

```yaml
telemetrystore:
  spec: { ... }
  <name>:
    # Common knobs — chart values
    resources:
      requests:
        cpu: "2"
    persistence:
      size: 100Gi
      storageClass: gp3
    # Escape hatch — additional values files
    patches:
    - path: values/telemetrystore-production.yaml
```

### Docker Compose casting (output = compose.yaml)

Common knobs use compose service-level concepts.

`patches` for docker casting points to compose override files:

```yaml
signoz:
  spec: { ... }
  <name>:
    memLimit: 4g
    cpus: "2"
    restart: unless-stopped
    patches:
    - path: compose.override.yaml
```

### Systemd casting (output = .service files)

Common knobs use systemd unit directive names.

`patches` for systemd casting points to drop-in files:

```yaml
signoz:
  spec: { ... }
  <name>:
    memoryMax: 4G
    cpuQuota: 200%
    restartSec: 10
    patches:
    - path: signoz.service.d/override.conf
```

## Design process for casting designers

When building the platform field for a new casting, follow these steps:

### Step 1: Does the output format have a user-facing config layer?

- **Yes** (helm values, fly.toml) → Common knobs pass through as that layer's values. No foundry-invented keys.
- **No** (raw K8s manifests, compose.yaml, systemd units) → Define flattened common knobs using the platform's field names.

### Step 2: Inventory the output surfaces

List every generated resource per component and its tuning points (scheduling, compute, storage, networking, metadata).

### Step 3: Classify against what foundry already owns

| Already in MoldingSpec | Common knob candidate |
| --- | --- |
| replicas, shards (Cluster) | tolerations, nodeSelector |
| image, version (Image, Version) | cpu/memory requests/limits |
| app env vars (Env) | storageClass, storageSize |
| app config files (Config.Data) | service type, annotations |

Rule: if foundry's pipeline needs to read it (enricher, molding, or merge), it belongs in `spec`. If only the template reads it, it's a common knob.

### Step 4: Pick 5-10 flattened keys for common cases

Name them using the platform's native terminology (see kustomization schema naming conventions). These cover 90% of users.

### Step 5: Define the `patches` behavior

Determine how `patches` maps to the platform's native customization mechanism:

- **Kustomize**: patches become entries in `kustomization.yaml` `patches:` field
- **Helm**: patches become additional `--values` files
- **Docker**: patches become `compose.override.yaml` references
- **Systemd**: patches become `.service.d/` drop-in files

The `Patch` struct should support both inline content (`patch`) and external file references (`path`).

### Step 6: Write the typed validation struct

In the casting package (not api/v1alpha1). This struct serves three purposes: validation, schema generation, and documentation. It must include both common knobs and the `Patches` field.

### Step 7: Define the template contract

For each common knob: where it lands in the output, what the default is, which components support it. For patches: how they're injected into the generated output.

## Patches: platform-native escape hatch

The `patches` field inside each component's platform field is the escape hatch for anything the common knobs don't cover. It supports two forms:

### Inline patches

The user writes platform-native patch content directly in casting.yaml:

```yaml
signoz:
  <name>:
    patches:
    - patch: |
        apiVersion: apps/v1
        kind: StatefulSet
        metadata:
          name: signoz-signoz
        spec:
          template:
            spec:
              topologySpreadConstraints:
              - maxSkew: 1
                topologyKey: topology.kubernetes.io/zone
      target:
        kind: StatefulSet
        name: signoz-signoz
```

### Path-based patches

The user references external patch files. These files are platform-native and live alongside `casting.yaml`:

```yaml
signoz:
  <name>:
    patches:
    - path: patches/signoz-topology.yaml
      target:
        kind: StatefulSet
```

```text
project/
├── casting.yaml
├── patches/
│   ├── signoz-topology.yaml        ← topologySpreadConstraints, etc.
│   └── telemetrystore-pdb.yaml     ← PodDisruptionBudget patch
└── pours/
    └── deployment/                  ← foundry generates this (base)
```

### How patches flow per casting

| Casting | What `patches` becomes in the output |
| --- | --- |
| Kustomize | Entries in component `kustomization.yaml` `patches:` field |
| Helm | Additional `--values` file arguments |
| Docker | `compose.override.yaml` references |
| Systemd | `.service.d/` drop-in file references |

For kustomize casting, foundry doesn't need to build a patch engine — **kustomize IS the patch engine**. The `patches` from casting.yaml are injected into the generated `kustomization.yaml` files, and `kustomize build` applies them natively.

### `path` vs inline `patch` — when to use which

| Use case | Recommended | Why |
| --- | --- | --- |
| Simple, single-resource patch | Inline `patch` | Keeps everything in casting.yaml |
| Large or multi-resource patches | `path` reference | Avoids YAML-in-YAML clutter |
| Shared across components | `path` reference | Single file, referenced by multiple components |
| Sensitive configuration | `path` reference | Can be .gitignored separately |

## Future: common knobs as patches

Today, common knobs are handled via template conditionals (`{{ if }}`). As the number of knobs grows (5 components × 10 keys = 50 conditionals), templates get cluttered.

A future internal refactor could translate common knobs into patches automatically — for kustomize casting, foundry would generate kustomize patches from keys like `tolerations` and `resources`, eliminating template conditionals entirely.

**This is a non-breaking internal refactor.** The casting.yaml schema stays identical — same field names, same YAML structure. Users see no change. The `patches` field already uses the platform's native patch format, so the translation is straightforward.

## Relationship to K8s API conventions

Foundry's casting.yaml is a **CLI configuration file**, not a Kubernetes API object. K8s API conventions (api-conventions.md) are designed for objects served by an API server with CRUD operations, versioning, watch semantics, and strategic merge patch support. Casting.yaml is read once by `foundryctl`.

The relevant K8s parallel is **kubectl generator conventions** (kubectl-conventions.md): "Generators are kubectl commands that generate resources based on a set of inputs." That's exactly what `foundryctl forge` is.

What we borrow from K8s conventions:
- **Spec/Status separation** — spec = desired state, status = computed state. Foundry follows this.
- **Declarative field names** — fields describe desired state, not actions. Our fields (`tolerations`, `resources`, `storageSize`) are all declarative nouns.
- **camelCase field names, no abbreviations, no dashes** — consistent with K8s naming.
- **Optional fields use omitempty** — overrides are entirely optional.

What we intentionally deviate from:
- **`map[string]any` for override values** — K8s requires fully typed fields. We accept this deviation because the override struct in the casting package provides validation, and casting.yaml is not a K8s API object.
- **Separate field for platform tuning** — K8s puts everything in `spec`. We separate because foundry's `spec` is cross-platform, and platform tuning is casting-specific. This is a deliberate architectural choice, not an oversight.

## Naming (OPEN — to be finalized)

The per-component platform field name is not yet finalized.

### Naming insights from the kustomization schema

The [kustomization.json schema](https://github.com/SchemaStore/schemastore/blob/master/src/schemas/json/kustomization.json) provides naming conventions worth following:

1. **Every field is a declarative noun** — `patches`, `replicas`, `images`, `resources`, `replacements`, `commonLabels`. Not verbs, not actions. They describe what the user wants, not what the system should do.
2. **Platform-native terminology** — kustomize uses K8s vocabulary (`patches`, `namespace`, `labels`), not its own invented terms.
3. **No generic wrapper** — kustomize doesn't have an `overrides` or `customization` bag. Each concept has its own first-class field name.
4. **The tool name itself is a noun** — "Kustomization" = "the customization." The entire file IS the desired state.

### What we need

The field wraps both common knobs (platform-native keys like `tolerations`, `resources`) and `patches` (the escape hatch). It sits alongside `spec` on each component:

```yaml
signoz:
  spec: { ... }       # foundry domain — what the component is
  <name>:              # platform domain — how the component runs
    tolerations: [...]
    resources: { ... }
    patches: [...]
```

The field is **desired state about HOW the component runs on the chosen platform**. It is NOT:

- "Overrides" — they're not corrections to defaults, they're desired state
- "Tuning" — implies adjustment/optimization, not declaration
- A verb — K8s conventions require declarative noun names

### Candidates reconsidered

| Name | Pros | Cons |
| --- | --- | --- |
| `overrides` | KRM-familiar, clear intent | Implies "fighting defaults" — but these ARE desired state, not corrections |
| `deploy` | Clear, noun form works | Overlaps with `spec.deployment` |
| `runtime` | Describes HOW the component runs | K8s uses this for container runtime, potential confusion |
| `platform` | Describes what it's for | Too generic, doesn't convey "desired state" |
| `placement` | KRM-aligned (scheduling concerns) | Too narrow — covers scheduling but not resources, storage, services |
| `infra` | Short, clear | Abbreviation (K8s convention: no abbreviations) |
| `infrastructure` | Declarative noun, describes the domain | Verbose but precise; the field IS infrastructure desired state |

Key constraint from K8s conventions: "fields in the specification should have declarative rather than imperative names — they represent the desired state, not actions intended to yield the desired state."

## User personas

### Simple user
"I want SigNoz running. I don't care about K8s details."

Uses defaults. Maybe sets `replicas: 3`. Never touches overrides. `foundryctl forge && foundryctl cast` and done.

### Platform engineer
"I need SigNoz on our K8s cluster with specific tolerations for our node pools, resource limits for cost control, and gp3 storage."

Knows K8s well. Uses common knobs with K8s field names for typical needs. Drops to `patches` for advanced scheduling (topologySpreadConstraints, pod disruption budgets). Runs `foundryctl cast` for end-to-end deployment.

### SigNoz maintainer
"I need to test template changes and validate the platform field works correctly."

Uses maintainer-mode annotations. Tests locally with `foundryctl forge`. Validates generated manifests before publishing.

### Casting designer
"I'm building a new casting (e.g., Nomad, ECS) and need to define what common knobs and patches my casting supports."

Follows the 6-step design process above. Defines a typed validation struct (including `Patch` types). Writes clean templates. Documents supported keys and patch behavior.

## Migration path

The platform field is additive — it's a new optional field on each component type. Existing casting.yaml files without it continue to work exactly as before. No breaking changes.

When common knobs are internally translated to patches (future refactor), it's an internal change behind `Forge()`. The casting.yaml schema is unchanged.
