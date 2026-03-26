ARG SIGNOZ_VERSION=latest
ARG INGESTER_VERSION=latest
ARG FOUNDRY_VERSION=latest

# Stage 1: Download all binaries
FROM alpine:3.21 AS downloader

ARG TARGETARCH
ARG SIGNOZ_VERSION
ARG INGESTER_VERSION
ARG FOUNDRY_VERSION

RUN apk add --no-cache curl tar

# SigNoz
RUN mkdir -p /out/opt/signoz && \
    curl -fsSL "https://github.com/SigNoz/signoz/releases/${SIGNOZ_VERSION}/download/signoz_linux_${TARGETARCH}.tar.gz" | \
        tar -xz --strip-components=1 -C /out/opt/signoz && \
    chmod +x /out/opt/signoz/bin/signoz

# OTel Collector (ingester)
RUN mkdir -p /out/opt/ingester && \
    curl -fsSL "https://github.com/SigNoz/signoz-otel-collector/releases/${INGESTER_VERSION}/download/signoz-otel-collector_linux_${TARGETARCH}.tar.gz" | \
        tar -xz --strip-components=1 -C /out/opt/ingester && \
    chmod +x /out/opt/ingester/bin/signoz-otel-collector

# Foundry
RUN mkdir -p /out/usr/local/bin && \
    curl -fsSL "https://github.com/SigNoz/foundry/releases/${FOUNDRY_VERSION}/download/foundry_linux_${TARGETARCH}.tar.gz" | \
        tar -xz --strip-components=1 -C /tmp && \
    mv /tmp/bin/foundryctl /out/usr/local/bin/foundryctl && \
    chmod +x /out/usr/local/bin/foundryctl

# Stage 2: Runtime with systemd
FROM debian:bookworm-slim

# Install systemd
RUN apt-get update && apt-get install -y --no-install-recommends \
        systemd \
        systemd-sysv \
        ca-certificates \
        curl \
        gnupg && \
    rm -f /lib/systemd/system/multi-user.target.wants/* \
          /etc/systemd/system/*.wants/* \
          /lib/systemd/system/local-fs.target.wants/* \
          /lib/systemd/system/sockets.target.wants/*udev* \
          /lib/systemd/system/sockets.target.wants/*initctl* \
          /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
          /lib/systemd/system/systemd-update-utmp*

# Install ClickHouse from apt repo
RUN curl -fsSL 'https://packages.clickhouse.com/rpm/lts/repodata/repomd.xml.key' | \
        gpg --dearmor -o /usr/share/keyrings/clickhouse-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/clickhouse-keyring.gpg] https://packages.clickhouse.com/deb stable main" \
        > /etc/apt/sources.list.d/clickhouse.list && \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        clickhouse-server \
        clickhouse-client && \
    systemctl disable clickhouse-server || true

RUN apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --from=downloader /out/ /

RUN useradd -r -m -d /opt/signoz -s /bin/false signoz && \
    mkdir -p /var/lib/signoz /var/lib/ingester /etc/foundry /opt/signoz/web && \
    chown signoz:signoz /var/lib/signoz /var/lib/ingester /opt/signoz

COPY docs/examples/standalone/casting.yaml /etc/foundry/casting.yaml
COPY docs/examples/standalone/foundry-setup.service /etc/systemd/system/foundry-setup.service
RUN mkdir -p /var/lib/foundry && systemctl enable foundry-setup.service

EXPOSE 8080 4317 4318
VOLUME ["/var/lib/clickhouse", "/var/lib/signoz"]

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
