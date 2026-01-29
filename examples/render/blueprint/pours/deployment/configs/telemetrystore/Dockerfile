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