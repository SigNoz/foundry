# Helm with Knobs

Uses `config.knobs` to tune Helm values without writing raw values overrides.

This example demonstrates:

- Setting CPU/memory resources and persistent storage for ClickHouse (telemetrystore)
- Scheduling constraints via tolerations and nodeSelector
- Pod annotations for Prometheus scraping on the ingester
- LoadBalancer service type with AWS NLB annotations for SigNoz UI

## Usage

```bash
foundryctl forge -f casting.yaml
```
