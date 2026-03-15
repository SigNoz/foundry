# Kustomize with Patches

Uses `patches` to add custom Kubernetes resources that knobs don't cover.

This example demonstrates:

- Adding an Ingress resource with TLS termination and cert-manager integration

Patch files live in `patches/` and are referenced by path in `casting.yaml`. Foundry resolves the paths relative to the kustomization output directory automatically.

## Usage

```bash
foundryctl forge -f casting.yaml
```
