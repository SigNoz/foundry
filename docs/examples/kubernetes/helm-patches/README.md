# Helm with Patches

Uses `patches` to inject additional Helm values that knobs don't cover.

This example demonstrates:

- Adding an Ingress configuration with TLS and cert-manager integration
- Enabling ServiceMonitor resources for ClickHouse and OTel Collector

Patch files live in `patches/` and contain raw Helm values that get merged into the generated `values.yaml`.

## Usage

```bash
foundryctl forge -f casting.yaml
```
