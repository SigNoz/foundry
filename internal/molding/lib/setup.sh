#!/bin/bash
# castings/linux/binary/standard/setup.sh

# ----------------------------------------------------
# PREAMBLE & HELPERS
# ----------------------------------------------------
BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
MOLDINGS_DIR="$BASE_DIR/../../../../moldings"
POURS_DIR="$BASE_DIR/pours"
LIB_DIR="$MOLDINGS_DIR/lib"
source "$LIB_DIR/forge.sh"

REQUIRED_VARS=(
    "ZOOKEEPER_VERSION" "SIGNOZ_VERSION" "OTEL_VERSION" "MIGRATOR_VERSION"
    "ZK_INSTALL_DIR" "SIGNOZ_INSTALL_DIR" "OTEL_INSTALL_DIR"
    "CLICKHOUSE_HOST" "CLICKHOUSE_TCP_PORT"
)

# Helper to detect architecture (amd64 vs arm64) as per docs
function get_arch() {
    uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g'
}

# ----------------------------------------------------
# ACTION: FORGE (Plan)
# ----------------------------------------------------
function cmd_forge() {
    log_info "🔨 [FORGE] Starting..."
    rm -rf "$POURS_DIR" && mkdir -p "$POURS_DIR"

    load_and_validate_env "$BASE_DIR/.env" "${REQUIRED_VARS[@]}"

    forge_component "zookeeper" "$MOLDINGS_DIR" "$POURS_DIR"
    forge_component "clickhouse" "$MOLDINGS_DIR" "$POURS_DIR"
    forge_component "signoz" "$MOLDINGS_DIR" "$POURS_DIR"
    forge_component "otel-collector" "$MOLDINGS_DIR" "$POURS_DIR"

    # Forge Systemd Units
    log_info "Forging Systemd units..."
    mkdir -p "$POURS_DIR/systemd"
    for unit in systemd/*.service; do
        envsubst < "$unit" > "$POURS_DIR/systemd/$(basename "$unit")"
    done
    
    log_succ "Artifacts ready in ./pours"
}

# ----------------------------------------------------
# ACTION: CAST (Apply)
# ----------------------------------------------------
function cmd_cast() {
    # 1. Validation & Prep
    cmd_forge
    log_info "🏗️  [CAST] Deploying SigNoz to Linux..."
    if [ "$EUID" -ne 0 ]; then log_err "Run as root"; exit 1; fi
    load_and_validate_env "$BASE_DIR/.env" "${REQUIRED_VARS[@]}"
    ARCH=$(get_arch)

    # 2. PREREQUISITES (Java)
    log_info "Step 1: Prerequisites (Java)..."
    if ! command -v java &> /dev/null; then
        apt update && apt install default-jdk -y
    fi

    # 3. CLICKHOUSE
    log_info "Step 2: Installing ClickHouse..."
    if ! command -v clickhouse-server &> /dev/null; then
        curl -fsSL 'https://builds.clickhouse.com/master/install.sh' | bash
    fi
    # Configure CH
    mkdir -p /etc/clickhouse-server/config.d/
    cp "$POURS_DIR/clickhouse/cluster.xml" /etc/clickhouse-server/config.d/
    chown -R clickhouse:clickhouse /etc/clickhouse-server
    # Don't start yet, need ZK first

    # 4. ZOOKEEPER
    log_info "Step 3: Installing Zookeeper..."
    # Download
    if [ ! -d "$ZK_INSTALL_DIR/bin" ]; then
        cd /tmp
        curl -L "https://dlcdn.apache.org/zookeeper/zookeeper-${ZOOKEEPER_VERSION}/apache-zookeeper-${ZOOKEEPER_VERSION}-bin.tar.gz" -o zookeeper.tar.gz
        tar -xzf zookeeper.tar.gz
        mkdir -p "$ZK_INSTALL_DIR" "$ZK_DATA_DIR" "$ZK_LOG_DIR"
        cp -r apache-zookeeper-${ZOOKEEPER_VERSION}-bin/* "$ZK_INSTALL_DIR"
        rm zookeeper.tar.gz
    fi

    # Configs from Pours
    mkdir -p "$ZK_INSTALL_DIR/conf"
    cp "$POURS_DIR/zookeeper/zoo.cfg" "$ZK_INSTALL_DIR/conf/"
    cp "$POURS_DIR/zookeeper/zoo.env" "$ZK_INSTALL_DIR/conf/"
    
    # User & Perms
    id -u zookeeper &>/dev/null || useradd --system --home "$ZK_INSTALL_DIR" --no-create-home --user-group --shell /sbin/nologin zookeeper
    chown -R zookeeper:zookeeper "$ZK_INSTALL_DIR" "$ZK_DATA_DIR" "$ZK_LOG_DIR"
    
    # Service
    cp "$POURS_DIR/systemd/zookeeper.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now zookeeper
    
    # Now start ClickHouse
    log_info "Starting ClickHouse..."
    systemctl enable --now clickhouse-server

    # 5. SCHEMA MIGRATION
    log_info "Step 4: Running Schema Migrations..."
    cd /tmp
    curl -L "https://github.com/SigNoz/signoz-otel-collector/releases/download/${MIGRATOR_VERSION}/signoz-schema-migrator_linux_${ARCH}.tar.gz" -o migrator.tar.gz
    tar -xzf migrator.tar.gz
    MIG_BIN="./signoz-schema-migrator_linux_${ARCH}/bin/signoz-schema-migrator"
    
    # Run Sync & Async
    $MIG_BIN sync --dsn="tcp://${CLICKHOUSE_HOST}:${CLICKHOUSE_TCP_PORT}?password=${CLICKHOUSE_PASSWORD}" --replication=true
    $MIG_BIN async --dsn="tcp://${CLICKHOUSE_HOST}:${CLICKHOUSE_TCP_PORT}?password=${CLICKHOUSE_PASSWORD}" --replication=true
    rm migrator.tar.gz

    # 6. SIGNOZ
    log_info "Step 5: Installing SigNoz..."
    if [ ! -d "$SIGNOZ_INSTALL_DIR/bin" ]; then
        cd /tmp
        curl -L "https://github.com/SigNoz/signoz/releases/download/${SIGNOZ_VERSION}/signoz_linux_${ARCH}.tar.gz" -o signoz.tar.gz
        tar -xzf signoz.tar.gz
        mkdir -p "$SIGNOZ_INSTALL_DIR" "$SIGNOZ_DATA_DIR"
        cp -r signoz_linux_${ARCH}/* "$SIGNOZ_INSTALL_DIR"
        rm signoz.tar.gz
    fi

    # Configs
    mkdir -p "$SIGNOZ_INSTALL_DIR/conf"
    cp "$POURS_DIR/signoz/systemd.env" "$SIGNOZ_INSTALL_DIR/conf/"
    
    # User & Perms
    id -u signoz &>/dev/null || useradd --system --home "$SIGNOZ_INSTALL_DIR" --no-create-home --user-group --shell /sbin/nologin signoz
    chown -R signoz:signoz "$SIGNOZ_INSTALL_DIR" "$SIGNOZ_DATA_DIR"
    
    # Service
    cp "$POURS_DIR/systemd/signoz.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now signoz

    # 7. OTEL COLLECTOR
    log_info "Step 6: Installing OTel Collector..."
    if [ ! -d "$OTEL_INSTALL_DIR/bin" ]; then
        cd /tmp
        curl -L "https://github.com/SigNoz/signoz-otel-collector/releases/download/${OTEL_VERSION}/signoz-otel-collector_linux_${ARCH}.tar.gz" -o otel.tar.gz
        tar -xzf otel.tar.gz
        mkdir -p "$OTEL_INSTALL_DIR" "$OTEL_DATA_DIR"
        cp -r signoz-otel-collector_linux_${ARCH}/* "$OTEL_INSTALL_DIR"
        rm otel.tar.gz
    fi

    # Configs
    mkdir -p "$OTEL_INSTALL_DIR/conf"
    cp "$POURS_DIR/otel-collector/"* "$OTEL_INSTALL_DIR/conf/"
    
    # Perms
    chown -R signoz:signoz "$OTEL_INSTALL_DIR" "$OTEL_DATA_DIR"
    
    # Service
    cp "$POURS_DIR/systemd/signoz-otel-collector.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now signoz-otel-collector

    log_succ "Installation Complete. Check status via systemctl."
}

# ----------------------------------------------------
# ACTION: SCRAP (Destroy)
# ----------------------------------------------------
function cmd_scrap() {
    log_info "[SCRAP] Melting down deployment..."
    systemctl stop signoz-otel-collector signoz zookeeper clickhouse-server
    
    # Optional: Remove dirs? Warn user first?
    # rm -rf /opt/signoz /opt/zookeeper ...
    
    log_succ "Services stopped."
}

case "${1:-}" in
    forge) cmd_forge ;;
    cast)  cmd_cast ;;
    scrap) cmd_scrap ;;
    *)     echo "Usage: ./setup.sh {forge|cast|scrap}"; exit 1 ;;
esac