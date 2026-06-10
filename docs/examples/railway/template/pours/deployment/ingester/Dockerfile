FROM signoz/signoz-otel-collector:latest

COPY ingester.yaml /etc/otel-collector-config.yaml
COPY opamp.yaml /etc/opamp-config.yaml


ENV SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_DSN=tcp://signoz-telemetrystore-clickhouse.railway.internal:9000

ENV LOW_CARDINAL_EXCEPTION_GROUPING=false
ENV SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_CLUSTER=cluster
ENV SIGNOZ_OTEL_COLLECTOR_CLICKHOUSE_REPLICATION=true
ENV SIGNOZ_OTEL_COLLECTOR_TIMEOUT=10m