# Kustomize with Knobs

Uses `config.knobs` to tune Kubernetes resource parameters without writing raw manifests.

This example demonstrates:
- Setting CPU/memory resources and storage for ClickHouse (telemetrystore)
- Scheduling constraints via tolerations and nodeSelector
- Pod annotations for Prometheus scraping on the ingester
- LoadBalancer service type with AWS NLB annotations for SigNoz UI

## Usage

```bash
foundryctl forge -f casting.yaml
```
