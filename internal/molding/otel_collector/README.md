# OpenTelemetry Collector - Composable Configuration

A fully composable, platform-agnostic OpenTelemetry Collector configuration for SigNoz, built with CUE.

## Features

- **Fully Composable**: Every field can be modified, added, or removed
- **Platform Agnostic**: No platform-specific logic - compose flavors on top
- **Type Safe**: CUE provides strong typing and validation
- **Sensible Defaults**: Works out of the box with reasonable defaults

## Files

- `config.cue` - Main composable configuration
- `example-usage.cue` - Comprehensive examples of different use cases
- `otel-collector-config.yaml` - Original YAML config (reference)

## Quick Start

### 1. Basic Export

Create a file to set required values:

```bash
cat > my-config.cue << 'EOF'
package otel_collector

config: clickhouse_host: "clickhouse"
EOF
```

Export to YAML:

```bash
cue export config.cue my-config.cue -e receivers -e processors -e extensions -e exporters -e service --out yaml > otel-config.yaml
```

### 2. Using a Flavor

Create a high-throughput production flavor:

```bash
cat > prod-high-throughput.cue << 'EOF'
package otel_collector

config: {
    clickhouse_host: "clickhouse-prod.internal"
    batch_send_size: 100000
    batch_max_size: 110000
    batch_timeout: "60s"
    dimensions_cache_size: 1000000
    metrics_flush_interval: "30s"
}
EOF
```

Export:

```bash
cue export config.cue prod-high-throughput.cue -e receivers -e processors -e extensions -e exporters -e service --out yaml > prod-config.yaml
```

## Configuration Examples

### Modify Histogram Buckets

```cue
package otel_collector

config: {
    clickhouse_host: "clickhouse"
    latency_histogram_buckets: [
        "10us", "50us", "100us", "500us", "1ms", "5ms",
        "10ms", "50ms", "100ms", "500ms", "1s", "5s", "10s",
    ]
}
```

### Add Custom Dimensions

```cue
package otel_collector

config: {
    clickhouse_host: "clickhouse"
    dimensions: [
        {name: "service.namespace", default: "production"},
        {name: "deployment.environment", default: "prod"},
        {name: "signoz.collector.id"},
        {name: "custom.team"},
        {name: "custom.region"},
    ]
}
```

### Add Custom Pipeline

```cue
package otel_collector

config: clickhouse_host: "clickhouse"

service: pipelines: "metrics/custom": {
    receivers: ["otlp", "prometheus"]
    processors: ["batch", "resourcedetection"]
    exporters: ["signozclickhousemetrics"]
}
```

### Remove Default Pipelines

```cue
package otel_collector

config: clickhouse_host: "clickhouse"

// Override entire service to only include traces
service: {
    telemetry: logs: encoding: "json"
    extensions: ["health_check"]
    pipelines: {
        traces: {
            receivers: ["otlp"]
            processors: ["batch"]
            exporters: ["clickhousetraces"]
        }
    }
}
```

### Development Minimal Flavor

```cue
package otel_collector

config: {
    clickhouse_host: "localhost"
    batch_send_size: 100
    batch_max_size: 200
    batch_timeout: "1s"
    low_cardinal_exception_grouping: true
    telemetry_log_encoding: "console"
}
```

## Testing Commands

### Validate Configuration

```bash
# Allow incomplete instances (since clickhouse_host is required)
cue vet -c=false config.cue
```

### Test with Examples

```bash
# Export example 1 (basic modification)
cue export config.cue -t example=example1 --out yaml

# Test production high throughput flavor
cat > test-prod.cue << 'EOF'
package otel_collector
config: {
    clickhouse_host: "clickhouse-prod"
    batch_send_size: 100000
    batch_max_size: 110000
}
EOF

cue export config.cue test-prod.cue -e receivers -e processors -e extensions -e exporters -e service --out yaml
```

### Verify Specific Sections

```bash
# Export only receivers
cue export config.cue my-config.cue -e receivers --out yaml

# Export only processors
cue export config.cue my-config.cue -e processors --out yaml

# Export only service configuration
cue export config.cue my-config.cue -e service --out yaml
```

### Test Custom Dimensions

```bash
cat > test-dimensions.cue << 'EOF'
package otel_collector

config: {
    clickhouse_host: "clickhouse"
    dimensions: [
        {name: "service.name"},
        {name: "custom.team"},
        {name: "custom.datacenter", default: "us-east-1"},
    ]
}
EOF

cue export config.cue test-dimensions.cue -e processors --out yaml | grep -A 20 dimensions
```

### Test Custom Histogram Buckets

```bash
cat > test-buckets.cue << 'EOF'
package otel_collector

config: {
    clickhouse_host: "clickhouse"
    latency_histogram_buckets: ["1ms", "10ms", "100ms", "1s", "10s"]
}
EOF

cue export config.cue test-buckets.cue -e processors --out yaml | grep -A 10 latency_histogram_buckets
```

### Test Adding Pipelines

```bash
cat > test-pipelines.cue << 'EOF'
package otel_collector

config: clickhouse_host: "clickhouse"

service: pipelines: "traces/custom": {
    receivers: ["otlp"]
    processors: ["resourcedetection", "batch"]
    exporters: ["clickhousetraces"]
}
EOF

cue export config.cue test-pipelines.cue -e service --out yaml
```

## Available Configuration Parameters

### Required
- `clickhouse_host` - ClickHouse host address

### Optional (with defaults)

**Receivers:**
- `otlp_grpc_endpoint` (default: "0.0.0.0:4317")
- `otlp_http_endpoint` (default: "0.0.0.0:4318")
- `prometheus_scrape_interval` (default: "60s")
- `prometheus_scrape_targets` (default: ["localhost:8888"])

**Processors:**
- `batch_send_size` (default: 10000)
- `batch_max_size` (default: 11000)
- `batch_timeout` (default: "10s")
- `resource_detectors` (default: ["env", "system"])
- `resource_timeout` (default: "2s")

**Span Metrics:**
- `metrics_exporter` (default: "signozclickhousemetrics")
- `metrics_flush_interval` (default: "60s")
- `latency_histogram_buckets` (default: 17 predefined buckets)
- `dimensions_cache_size` (default: 100000)
- `aggregation_temporality` (default: "AGGREGATION_TEMPORALITY_DELTA")
- `enable_exp_histogram` (default: true)
- `dimensions` (default: 12 standard dimensions)

**Extensions:**
- `health_check_endpoint` (default: "0.0.0.0:13133")
- `pprof_endpoint` (default: "0.0.0.0:1777")

**Exporters:**
- `clickhouse_port` (default: 9000)
- `clickhouse_timeout` (default: "10s")
- `use_new_schema` (default: true)
- `low_cardinal_exception_grouping` (default: false)

**Telemetry:**
- `telemetry_log_encoding` (default: "json")

## Integration with Castings

In your platform-specific castings (Docker, Kubernetes, Linux), import and compose:

```cue
package myplatform

import otel "path/to/moldings/otel-collector"

// Set required and platform-specific values
otel.config: {
    clickhouse_host: "clickhouse.namespace.svc.cluster.local"
    prometheus_scrape_targets: ["otel-collector-0:8888", "otel-collector-1:8888"]
}

// Optionally modify pipelines or other sections
otel.service: pipelines: "metrics/k8s": {
    receivers: ["prometheus"]
    processors: ["batch", "resourcedetection"]
    exporters: ["signozclickhousemetrics"]
}
```

## Notes

- The configuration is designed to be composed, not forked
- Create flavors by overriding specific values, not by duplicating the entire config
- Use CUE's unification to merge your overrides with the base configuration
- Platform-specific logic belongs in castings/, not here in moldings/
