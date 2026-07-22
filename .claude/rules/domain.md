---
paths:
  - "internal/domain/**"
---

# `internal/domain` layer rules

## What this package is for
`internal/domain` holds Foundry's **shared domain primitives**: value objects
and pure helpers reused across configuration loading, molding, casting,
patching, and writing. It is imported almost everywhere (api/v1alpha1, every
casting, every molding, config, foundry, infrastructure, patch, writer, ledger,
cmd) — so it sits at the bottom of the dependency graph and must stay
dependency-light. Concretely it owns:

- **Materials** — the generated-file abstraction: structured (`JSONMaterial`,
  `YAMLMaterial`, `INIMaterial`) and opaque (`BlobMaterial`).
- **Addresses & formats** — `Address` and serialization helpers (`yaml.go`).
- **Template wrappers** (`template.go`), **pointer helpers** (`pointer.go`), and
  small value types (`event`, `properties`, `distinct_id`, `file`).

It is *not* where behavior lives. It must **not** decide which pipeline phase
runs, select deployment implementations, execute tools, write files, read user
configuration, or hold platform-specific rendering logic. That belongs in the
package that owns the behavior (molding, casting, writer, tooler, config, …).

## Import restrictions (see `internal/domain/doc.go`)
Domain may import **only** `internal/errors`. Do **not** import `api/v1alpha1`
or any orchestration / casting / molding / infrastructure / writer / tooler /
config / ledger package from here. Keep the package side-effect free.

## Materials
- `Material` is `Path() + FmtContents()`.
- `StructuredMaterial` adds `JSONContents`, `HasMultipleDocuments`,
  `CloneWithJSONContents`, and `GetBytes` / `GetStringSlice` (gjson dotted-path
  syntax, e.g. `service.names.0`).
- Concrete types: `JSONMaterial`, `YAMLMaterial` (multi-doc via
  `kio.ByteReader`/`ByteWriter`), `INIMaterial` (`gopkg.in/ini.v1`, sections
  sorted alphabetically for determinism), `BlobMaterial` (opaque).
- Constructors follow `NewX` / `MustNewX`.

## Address literals
Build address/port strings with
`domain.MustNewAddress("scheme", host, port).String()`. Never hand-format with
`Sprintf`.
