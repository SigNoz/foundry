FROM signoz/signoz-otel-collector:latest

COPY ingester.yaml /etc/otel-collector-config.yaml
COPY opamp.yaml /etc/opamp-config.yaml


ENV SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_DSN=tcp://signoz-telemetrystore-clickhouse.railway.internal:9000
