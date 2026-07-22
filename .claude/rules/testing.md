---
paths:
  - "**/*_test.go"
---

# Test conventions

- **Co-located, per-source test files**: `yaml_material_test.go` sits next to
  `yaml_material.go`. One test file per source file.
- **Table-driven** with a `pass bool` validity gate on each case.
- **Subtest names are `Topic_Status`**: `IPv6_Valid`, `MissingScheme_Invalid`.
- **Expected fields are prefixed `expected*`** (e.g. `expectedAddress`).

Run tests with the standard tooling:

```
go test ./...                                             # everything
go test ./internal/domain/...                             # one package tree
go test -run TestNewAddress ./internal/domain/            # single test
go test -run TestNewAddress/IPv6_Valid ./internal/domain/ # single subtest
```
