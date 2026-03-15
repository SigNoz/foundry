# Helm with Knobs and Patches

Combines `config.knobs` for platform tuning with `patches` for advanced Helm values customization.

This example demonstrates:

- Knobs for resource limits and scheduling (tolerations, nodeSelector) on telemetrystore and ingester
- A patch for additional Helm values: ClickHouse persistence, Ingress, and OTel Collector minReadySeconds

Use knobs for common operational parameters. Use patches when you need Helm values that knobs don't expose.

## Usage

```bash
foundryctl forge -f casting.yaml
```
