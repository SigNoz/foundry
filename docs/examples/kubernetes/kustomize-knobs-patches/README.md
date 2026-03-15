# Kustomize with Knobs and Patches

Combines `config.knobs` for platform tuning with `patches` for advanced customization.

This example demonstrates:

- Knobs for resource limits, storage, scheduling (tolerations, nodeSelector), service type, and pod annotations
- Patches for adding an Ingress with TLS and a PodDisruptionBudget for ClickHouse

Use knobs for common operational parameters. Use patches when you need resources or fields that knobs don't expose.

## Usage

```bash
foundryctl forge -f casting.yaml
```
