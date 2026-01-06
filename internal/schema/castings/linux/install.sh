#!/bin/bash

set -e
set -u
set -o pipefail

POURS_DIR="${POURS_DIR:-./pours}"
ZOOKEEPER_VERSION="3.8.5"
ZOOKEEPER_DIR="/opt/zookeeper"
ZOOKEEPER_DATA_DIR="/var/lib/zookeeper"
ZOOKEEPER_LOG_DIR="/var/log/zookeeper"
SIGNOZ_DIR="/opt/signoz"
SIGNOZ_DATA_DIR="/var/lib/signoz"
COLLECTOR_DIR="/opt/signoz-otel-collector"
COLLECTOR_DATA_DIR="/var/lib/signoz-otel-collector"

# Check if running as root
check_root() {
    [ "$EUID" -eq 0 ] || { echo "Please run as root"; exit 1; }
}

# Check if pours directory exists
check_pours_dir() {
    [ -d "$POURS_DIR" ] || { echo "Pours directory not found: $POURS_DIR"; exit 1; }
}

# Check if required file exists in pours
require_pours_file() {
    [ -f "$1" ] || { echo "Required file not found: $1"; exit 1; }
}

# Detect architecture
detect_arch() {
    uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g'
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Process template file (substitute variables)
process_template() {
    sed -e "s|\${ZOOKEEPER_INSTALL_DIR}|${ZOOKEEPER_DIR}|g" \
        -e "s|\${ZOOKEEPER_DATA_DIR}|${ZOOKEEPER_DATA_DIR}|g" \
        -e "s|\${ZOOKEEPER_LOG_DIR}|${ZOOKEEPER_LOG_DIR}|g" \
        "$1" > "$2"
}

check_root
check_pours_dir

# Install Java
if ! command_exists java; then
    if command_exists apt-get; then
        apt-get update && apt-get install -y default-jdk
    elif command_exists yum; then
        yum install -y java-1.8.0-openjdk java-1.8.0-openjdk-devel
    elif command_exists dnf; then
        dnf install -y java-1.8.0-openjdk java-1.8.0-openjdk-devel
    else
        echo "Package manager not found. Please install Java manually."
        exit 1
    fi
fi

# Install Zookeeper
cd /tmp
curl -L "https://dlcdn.apache.org/zookeeper/zookeeper-${ZOOKEEPER_VERSION}/apache-zookeeper-${ZOOKEEPER_VERSION}-bin.tar.gz" -o zookeeper.tar.gz
tar -xzf zookeeper.tar.gz
mkdir -p "${ZOOKEEPER_DIR}" "${ZOOKEEPER_DATA_DIR}" "${ZOOKEEPER_LOG_DIR}"
cp -r "apache-zookeeper-${ZOOKEEPER_VERSION}-bin"/* "${ZOOKEEPER_DIR}"
require_pours_file "${POURS_DIR}/zookeeper/zoo.cfg"
mkdir -p "${ZOOKEEPER_DIR}/conf"
cp "${POURS_DIR}/zookeeper/zoo.cfg" "${ZOOKEEPER_DIR}/conf/zoo.cfg"
require_pours_file "${POURS_DIR}/zookeeper/zoo.env"
cp "${POURS_DIR}/zookeeper/zoo.env" "${ZOOKEEPER_DIR}/conf/zoo.env"
if ! getent passwd zookeeper >/dev/null; then
    useradd --system --home "${ZOOKEEPER_DIR}" --no-create-home --user-group --shell /sbin/nologin zookeeper
fi
chown -R zookeeper:zookeeper "${ZOOKEEPER_DIR}" "${ZOOKEEPER_DATA_DIR}" "${ZOOKEEPER_LOG_DIR}"
require_pours_file "${POURS_DIR}/linux/zookeeper.service"
process_template "${POURS_DIR}/linux/zookeeper.service" /etc/systemd/system/zookeeper.service
systemctl daemon-reload
systemctl start zookeeper.service
systemctl enable zookeeper.service
rm -f /tmp/zookeeper.tar.gz
rm -rf "/tmp/apache-zookeeper-${ZOOKEEPER_VERSION}-bin"

# Configure ClickHouse
mkdir -p /etc/clickhouse-server/config.d
require_pours_file "${POURS_DIR}/clickhouse/cluster.xml"
cp "${POURS_DIR}/clickhouse/cluster.xml" /etc/clickhouse-server/config.d/cluster.xml
chown clickhouse:clickhouse /etc/clickhouse-server/config.d/cluster.xml
systemctl start clickhouse-server.service
systemctl enable clickhouse-server.service

# Run ClickHouse migrations
ARCH=$(detect_arch)
cd /tmp
curl -L "https://github.com/SigNoz/signoz-otel-collector/releases/latest/download/signoz-schema-migrator_linux_${ARCH}.tar.gz" -o signoz-schema-migrator.tar.gz
tar -xzf signoz-schema-migrator.tar.gz
MIGRATOR_BIN=$(find /tmp -name "signoz-schema-migrator" -type f | head -1)
[ -n "$MIGRATOR_BIN" ] || { echo "Could not find signoz-schema-migrator binary"; exit 1; }
chmod +x "$MIGRATOR_BIN"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-password}"
DSN="tcp://localhost:9000?password=${CLICKHOUSE_PASSWORD}"
"$MIGRATOR_BIN" sync --dsn="${DSN}" --replication=true --up=
"$MIGRATOR_BIN" async --dsn="${DSN}" --replication=true --up=
rm -f /tmp/signoz-schema-migrator.tar.gz
rm -rf /tmp/signoz-schema-migrator_linux_*

# Install PostgreSQL
if ! command_exists psql; then
    if command_exists apt-get; then
        apt-get update && apt-get install -y postgresql postgresql-contrib
    elif command_exists yum; then
        yum install -y postgresql-server postgresql-contrib
        postgresql-setup --initdb || true
    elif command_exists dnf; then
        dnf install -y postgresql-server postgresql-contrib
        postgresql-setup --initdb || true
    else
        echo "Package manager not found. Please install PostgreSQL manually."
        exit 1
    fi
fi

# Determine PostgreSQL data directory and version
if [ -d /var/lib/postgresql ]; then
    PGDATA="/var/lib/postgresql"
    PG_VERSION=$(ls -1 /var/lib/postgresql 2>/dev/null | head -1 || echo "main")
    PGDATA="${PGDATA}/${PG_VERSION}"
elif [ -d /var/lib/pgsql ]; then
    PGDATA="/var/lib/pgsql/data"
else
    PGDATA="/var/lib/postgresql/data"
fi

# Initialize database if not already initialized
if [ ! -d "$PGDATA" ] || [ -z "$(ls -A "$PGDATA" 2>/dev/null)" ]; then
    if command_exists apt-get; then
        sudo -u postgres /usr/lib/postgresql/*/bin/initdb -D "$PGDATA" || true
    else
        sudo -u postgres /usr/bin/initdb -D "$PGDATA" || true
    fi
fi

# Copy PostgreSQL configuration files
mkdir -p /etc/postgresql
require_pours_file "${POURS_DIR}/postgres/auth.env"
cp "${POURS_DIR}/postgres/auth.env" /etc/postgresql/postgresql.env

# Source auth.env to get credentials
. /etc/postgresql/postgresql.env

# Copy server config
if [ -f "${POURS_DIR}/postgres/serverConfig.conf" ]; then
    require_pours_file "${POURS_DIR}/postgres/serverConfig.conf"
    if [ -f "$PGDATA/postgresql.conf" ]; then
        cat "${POURS_DIR}/postgres/serverConfig.conf" >> "$PGDATA/postgresql.conf"
    fi
fi

# Copy HBA config
if [ -f "${POURS_DIR}/postgres/hbaConfig.conf" ] && [ -n "$(cat "${POURS_DIR}/postgres/hbaConfig.conf")" ]; then
    require_pours_file "${POURS_DIR}/postgres/hbaConfig.conf"
    cp "${POURS_DIR}/postgres/hbaConfig.conf" "$PGDATA/pg_hba.conf"
fi

# Set PGDATA in environment file
echo "PGDATA=$PGDATA" >> /etc/postgresql/postgresql.env

# Start PostgreSQL to create database and user
if command_exists systemctl; then
    systemctl start postgresql || service postgresql start || true
    sleep 2
fi

# Create database and user if they don't exist
sudo -u postgres psql -c "SELECT 1 FROM pg_roles WHERE rolname='${POSTGRES_USER:-signoz}'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE USER ${POSTGRES_USER:-signoz} WITH PASSWORD '${POSTGRES_PASSWORD:-Signoz@123}';" || true

sudo -u postgres psql -c "SELECT 1 FROM pg_database WHERE datname='${POSTGRES_DB:-signoz}'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE DATABASE ${POSTGRES_DB:-signoz} OWNER ${POSTGRES_USER:-signoz};" || true

# Grant privileges
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB:-signoz} TO ${POSTGRES_USER:-signoz};" || true

# Copy PostgreSQL service file if it exists
if [ -f "${POURS_DIR}/linux/postgres.service" ]; then
    require_pours_file "${POURS_DIR}/linux/postgres.service"
    cp "${POURS_DIR}/linux/postgres.service" /etc/systemd/system/postgresql.service
    systemctl daemon-reload
fi

# Enable and start PostgreSQL service
if command_exists systemctl; then
    systemctl enable postgresql.service || systemctl enable postgresql || true
    systemctl restart postgresql.service || systemctl restart postgresql || true
fi

# Install SigNoz
ARCH=$(detect_arch)
cd /tmp
curl -L "https://github.com/SigNoz/signoz/releases/latest/download/signoz_linux_${ARCH}.tar.gz" -o signoz.tar.gz
tar -xzf signoz.tar.gz
mkdir -p "${SIGNOZ_DIR}" "${SIGNOZ_DATA_DIR}" "${SIGNOZ_DIR}/conf"
cp -r "signoz_linux_${ARCH}"/* "${SIGNOZ_DIR}"
require_pours_file "${POURS_DIR}/linux/signoz.env"
cp "${POURS_DIR}/linux/signoz.env" "${SIGNOZ_DIR}/conf/systemd.env"
if ! getent passwd signoz >/dev/null; then
    useradd --system --home "${SIGNOZ_DIR}" --no-create-home --user-group --shell /sbin/nologin signoz
fi
chown -R signoz:signoz "${SIGNOZ_DATA_DIR}" "${SIGNOZ_DIR}"
require_pours_file "${POURS_DIR}/linux/signoz.service"
cp "${POURS_DIR}/linux/signoz.service" /etc/systemd/system/signoz.service
systemctl daemon-reload
systemctl start signoz.service
systemctl enable signoz.service
rm -f /tmp/signoz.tar.gz
rm -rf "/tmp/signoz_linux_${ARCH}"

# Install SigNoz Otel Collector
ARCH=$(detect_arch)
cd /tmp
curl -L "https://github.com/SigNoz/signoz-otel-collector/releases/latest/download/signoz-otel-collector_linux_${ARCH}.tar.gz" -o signoz-otel-collector.tar.gz
tar -xzf signoz-otel-collector.tar.gz
mkdir -p "${COLLECTOR_DATA_DIR}" "${COLLECTOR_DIR}" "${COLLECTOR_DIR}/conf"
cp -r "signoz-otel-collector_linux_${ARCH}"/* "${COLLECTOR_DIR}"
chown -R signoz:signoz "${COLLECTOR_DATA_DIR}" "${COLLECTOR_DIR}"
require_pours_file "${POURS_DIR}/signoz-otel-collector/config.yaml"
cp "${POURS_DIR}/signoz-otel-collector/config.yaml" "${COLLECTOR_DIR}/conf/config.yaml"
require_pours_file "${POURS_DIR}/linux/opamp.yaml"
cp "${POURS_DIR}/linux/opamp.yaml" "${COLLECTOR_DIR}/conf/opamp.yaml"
require_pours_file "${POURS_DIR}/linux/signoz-otel-collector.service"
cp "${POURS_DIR}/linux/signoz-otel-collector.service" /etc/systemd/system/signoz-otel-collector.service
systemctl daemon-reload
systemctl start signoz-otel-collector.service
systemctl enable signoz-otel-collector.service
rm -f /tmp/signoz-otel-collector.tar.gz
rm -rf "/tmp/signoz-otel-collector_linux_${ARCH}"
