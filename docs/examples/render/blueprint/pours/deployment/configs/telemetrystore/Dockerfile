FROM clickhouse/clickhouse-server:25.5.6

COPY config.d/ /etc/clickhouse-server/

RUN mkdir -p /var/lib/clickhouse/user_scripts/ \
    && chown clickhouse:clickhouse /var/lib/clickhouse/user_scripts

ARG VERSION=v0.0.1
# Download and install histogram-quantile binary
RUN cat > /tmp/install-histogram-quantile.sh <<'EOF'
#!/bin/bash
set -euo pipefail

node_os=$(uname -s | tr '[:upper:]' '[:lower:]')
node_arch=$(uname -m | sed 's/aarch64/arm64/' | sed 's/x86_64/amd64/')

echo "Fetching histogram-quantile for ${node_os}/${node_arch}"

cd /tmp
wget -O histogram-quantile.tar.gz \
  "https://github.com/SigNoz/signoz/releases/download/histogram-quantile%2F${VERSION}/histogram-quantile_${node_os}_${node_arch}.tar.gz"

tar -xzf histogram-quantile.tar.gz
mv histogram-quantile /var/lib/clickhouse/user_scripts/histogramQuantile
rm histogram-quantile.tar.gz

echo "histogram-quantile installed successfully"
EOF

RUN chmod +x /tmp/install-histogram-quantile.sh \
    && /tmp/install-histogram-quantile.sh \
    && rm /tmp/install-histogram-quantile.sh

# Create wrapper entrypoint to alias the right per-node config as config.yaml
RUN cat > /foundry-entrypoint.sh <<'EOF'
#!/bin/bash
set -euo pipefail

# Extract shard-replica indices from service name (e.g., signoz-telemetrystore-clickhouse-0-1)
INDICES=$(echo "${RENDER_SERVICE_NAME:-}" | grep -oE "[0-9]+-[0-9]+$" || echo "0-0")
CONFIG_SRC="/etc/clickhouse-server/config-${INDICES}.yaml"

if [ ! -f "$CONFIG_SRC" ]; then
  echo "Config $CONFIG_SRC not found, falling back to config-0-0.yaml"
  CONFIG_SRC="/etc/clickhouse-server/config-0-0.yaml"
fi

ln -sf "$CONFIG_SRC" /etc/clickhouse-server/config.yaml
echo "ClickHouse starting with config: $CONFIG_SRC (aliased as config.yaml)"

exec /entrypoint.sh "$@"
EOF

RUN chmod +x /foundry-entrypoint.sh

ENTRYPOINT ["/foundry-entrypoint.sh"]