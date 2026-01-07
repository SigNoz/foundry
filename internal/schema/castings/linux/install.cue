import "strings"
// InstallConfig defines configurable variables for the Linux install script
#InstallConfig: {
	// Component versions
	versions: {
		zookeeper: string | *"3.8.5"
		signoz: string | *"latest"
		clickhouse: string | *"25.5.6"
		signozOtelCollector: string | "latest"
		postgres: string | "latest"
		java:      string | *"default-jdk"  // For apt
	}

	// Installation directories
	directories: {
		zookeeper: {
			install: string | *"/opt/zookeeper"
			data:    string | *"/var/lib/zookeeper"
			log:     string | *"/var/log/zookeeper"
		}
		signoz: {
			install: string | *"/opt/signoz"
			data:    string | *"/var/lib/signoz"
		}
		collector: {
			install: string | *"/opt/signoz-otel-collector"
			data:    string | *"/var/lib/signoz-otel-collector"
		}
		clickhouse: {
			config: string | *"/etc/clickhouse-server"
		}
		postgres: {
			config: string | *"/etc/postgresql"
		}
	}

	// Download URLs (with version placeholders)
	urls: {
		zookeeper:       string | *"https://dlcdn.apache.org/zookeeper/zookeeper-${ZOOKEEPER_VERSION}/apache-zookeeper-${ZOOKEEPER_VERSION}-bin.tar.gz"
		signoz:          string | *"https://github.com/SigNoz/signoz/releases/${SIGNOZ_VERSION}/download/signoz_linux_${ARCH}.tar.gz"
		collector:       string | *"https://github.com/SigNoz/signoz-otel-collector/${SIGNOZ_OTEL_COLLECTOR_VERSION}/latest/download/signoz-otel-collector_linux_${ARCH}.tar.gz"
		schemaMigrator: string | *"https://github.com/SigNoz/signoz-otel-collector/${SIGNOZ_OTEL_COLLECTOR_VERSION}/download/signoz-schema-migrator_linux_${ARCH}.tar.gz"
	}

	// Retry configuration
	retries: {
		maxAttempts: int | *30
		sleepSec:    int | *2
	}

	// User/Group configuration
	users: {
		zookeeper: string | *"zookeeper"
		signoz:    string | *"signoz"
		clickhouse: string | *"clickhouse"
		postgres:  string | *"postgres"
	}
}

// InstallScript generates the complete install.sh from configuration
#InstallScript: {
	config: #InstallConfig

	// Generate bash variables section
	_variables: strings.Join([
		"# Component versions",
		"ZOOKEEPER_VERSION=\"\(config.versions.zookeeper)\"",
		"SIGNOZ_VERSION=\"v\(config.versions.signoz)\"",
		"SIGNOZ_OTEL_COLLECTOR_VERSION=\"v\(config.versions.signozOtelCollector)\"",
		"",
		"# Installation directories",
		"ZOOKEEPER_DIR=\"\(config.directories.zookeeper.install)\"",
		"ZOOKEEPER_DATA_DIR=\"\(config.directories.zookeeper.data)\"",
		"ZOOKEEPER_LOG_DIR=\"\(config.directories.zookeeper.log)\"",
		"SIGNOZ_DIR=\"\(config.directories.signoz.install)\"",
		"SIGNOZ_DATA_DIR=\"\(config.directories.signoz.data)\"",
		"COLLECTOR_DIR=\"\(config.directories.collector.install)\"",
		"COLLECTOR_DATA_DIR=\"\(config.directories.collector.data)\"",
		"CLICKHOUSE_CONFIG_DIR=\"\(config.directories.clickhouse.config)\"",
		"POSTGRES_CONFIG_DIR=\"\(config.directories.postgres.config)\"",
		"",
		"# Download URLs",
		"ZOOKEEPER_URL=\"\(config.urls.zookeeper)\"",
		"SIGNOZ_URL=\"\(config.urls.signoz)\"",
		"COLLECTOR_URL=\"\(config.urls.collector)\"",
		"SCHEMA_MIGRATOR_URL=\"\(config.urls.schemaMigrator)\"",
		"",
		"# Retry configuration",
		"MAX_RETRIES=\(config.retries.maxAttempts)",
		"RETRY_SLEEP=\(config.retries.sleepSec)",
		"",
		"# Users",
		"ZOOKEEPER_USER=\"\(config.users.zookeeper)\"",
		"SIGNOZ_USER=\"\(config.users.signoz)\"",
		"CLICKHOUSE_USER=\"\(config.users.clickhouse)\"",
		"POSTGRES_USER=\"\(config.users.postgres)\"",
	], "\n")

	_header: """
		#!/bin/bash

		set -e
		set -u
		set -o pipefail

		# Get the directory where this script is located
		SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

		# User can override with POURS_DIR environment variable
		POURS_DIR="${POURS_DIR:-$(cd "$SCRIPT_DIR/.." && pwd)}"

		"""

	_helperFunctions: """
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
		    sed -e "s|\\${ZOOKEEPER_INSTALL_DIR}|${ZOOKEEPER_DIR}|g" \\
		        -e "s|\\${ZOOKEEPER_DATA_DIR}|${ZOOKEEPER_DATA_DIR}|g" \\
		        -e "s|\\${ZOOKEEPER_LOG_DIR}|${ZOOKEEPER_LOG_DIR}|g" \\
		        "$1" > "$2"
		}

		# Wait for service to be ready with retries
		wait_for_service() {
		    local check_cmd="$1"
		    local service_name="$2"
		    local retry_count=0
		    
		    echo "Waiting for ${service_name} to be ready..."
		    until eval "$check_cmd"; do
		        retry_count=$((retry_count + 1))
		        if [ $retry_count -ge $MAX_RETRIES ]; then
		            echo "${service_name} failed to start after $MAX_RETRIES attempts"
		            exit 1
		        fi
		        echo "Waiting for ${service_name}... (attempt $retry_count/$MAX_RETRIES)"
		        sleep $RETRY_SLEEP
		    done
		    echo "${service_name} is ready"
		}

		check_root
		check_pours_dir

		"""

	_installJava: """
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

		"""

	_installZookeeper: """
		# Install Zookeeper
		cd /tmp
		curl -L "${ZOOKEEPER_URL}" -o zookeeper.tar.gz
		tar -xzf zookeeper.tar.gz
		mkdir -p "${ZOOKEEPER_DIR}" "${ZOOKEEPER_DATA_DIR}" "${ZOOKEEPER_LOG_DIR}"
		cp -r "apache-zookeeper-${ZOOKEEPER_VERSION}-bin"/* "${ZOOKEEPER_DIR}"
		require_pours_file "${POURS_DIR}/zookeeper/zoo.cfg"
		mkdir -p "${ZOOKEEPER_DIR}/conf"
		cp "${POURS_DIR}/zookeeper/zoo.cfg" "${ZOOKEEPER_DIR}/conf/zoo.cfg"
		if ! getent passwd "${ZOOKEEPER_USER}" >/dev/null; then
		    useradd --system --home "${ZOOKEEPER_DIR}" --no-create-home --user-group --shell /sbin/nologin "${ZOOKEEPER_USER}"
		fi
		chown -R "${ZOOKEEPER_USER}:${ZOOKEEPER_USER}" "${ZOOKEEPER_DIR}" "${ZOOKEEPER_DATA_DIR}" "${ZOOKEEPER_LOG_DIR}"
		require_pours_file "${POURS_DIR}/linux/zookeeper.service"
		process_template "${POURS_DIR}/linux/zookeeper.service" /etc/systemd/system/zookeeper.service
		systemctl daemon-reload
		systemctl start zookeeper.service
		systemctl enable zookeeper.service
		rm -f /tmp/zookeeper.tar.gz
		rm -rf "/tmp/apache-zookeeper-${ZOOKEEPER_VERSION}-bin"

		"""

	_installClickhouse: """
		# Install ClickHouse
		if ! command_exists clickhouse-server; then
		    if command_exists apt-get; then
		        apt-get install -y apt-transport-https ca-certificates curl gnupg
		        curl -fsSL 'https://packages.clickhouse.com/deb/repodata/repomd.xml.key' | gpg --dearmor -o /usr/share/keyrings/clickhouse-keyring.gpg
		        DEB_ARCH=$(dpkg --print-architecture)
		        echo "deb [signed-by=/usr/share/keyrings/clickhouse-keyring.gpg arch=${DEB_ARCH}] https://packages.clickhouse.com/deb stable main" | tee /etc/apt/sources.list.d/clickhouse.list
		        apt-get update
		        apt-get install -y clickhouse-server clickhouse-client
		    elif command_exists yum; then
		        yum install -y yum-utils
		        yum-config-manager --add-repo https://packages.clickhouse.com/rpm/clickhouse.repo
		        yum install -y clickhouse-server clickhouse-client
		    elif command_exists dnf; then
		        dnf install -y dnf-plugins-core
		        dnf config-manager --add-repo https://packages.clickhouse.com/rpm/clickhouse.repo
		        dnf install -y clickhouse-server clickhouse-client
		    else
		        echo "Package manager not found. Please install ClickHouse manually."
		        exit 1
		    fi
		fi

		# Configure ClickHouse
		mkdir -p "${CLICKHOUSE_CONFIG_DIR}/config.d"
		if ! getent passwd "${CLICKHOUSE_USER}" >/dev/null; then
		    useradd --system --shell /bin/false "${CLICKHOUSE_USER}" || true
		    chown "${CLICKHOUSE_USER}:${CLICKHOUSE_USER}" -R "${CLICKHOUSE_CONFIG_DIR}/"
		fi
		require_pours_file "${POURS_DIR}/clickhouse/config.yaml"
		cp "${POURS_DIR}/clickhouse/config.yaml" "${CLICKHOUSE_CONFIG_DIR}/config.yaml"
		chown "${CLICKHOUSE_USER}:${CLICKHOUSE_USER}" "${CLICKHOUSE_CONFIG_DIR}/config.yaml"
		require_pours_file "${POURS_DIR}/clickhouse/users.yaml"
		cp "${POURS_DIR}/clickhouse/users.yaml" "${CLICKHOUSE_CONFIG_DIR}/users.yaml"
		chown "${CLICKHOUSE_USER}:${CLICKHOUSE_USER}" "${CLICKHOUSE_CONFIG_DIR}/users.yaml"
		if [ -f "${POURS_DIR}/clickhouse/custom-function.yaml" ]; then
		    cp "${POURS_DIR}/clickhouse/custom-function.yaml" "${CLICKHOUSE_CONFIG_DIR}/custom-function.yaml"
		    chown "${CLICKHOUSE_USER}:${CLICKHOUSE_USER}" "${CLICKHOUSE_CONFIG_DIR}/custom-function.yaml"
		fi

		# Wait for Zookeeper to be ready
		wait_for_service 'curl -s -m 2 http://localhost:8080/commands/ruok | grep error | grep -q null' 'Zookeeper'

		echo "Starting Clickhouse Service"
		if require_pours_file "${POURS_DIR}/linux/clickhouse.service"; then
		    cp "${POURS_DIR}/linux/clickhouse.service" /etc/systemd/system/clickhouse.service
		    systemctl start clickhouse.service
		    systemctl enable clickhouse.service
		else
		    systemctl start clickhouse-server.service
		    systemctl enable clickhouse-server.service
		fi

		"""

	_runMigrations: """
		# Run ClickHouse migrations
		ARCH=$(detect_arch)
		cd /tmp
		curl -L "${SCHEMA_MIGRATOR_URL}" -o signoz-schema-migrator.tar.gz
		tar -xzf signoz-schema-migrator.tar.gz
		MIGRATOR_BIN=$(find /tmp -name "signoz-schema-migrator" -type f | head -1)
		[ -n "$MIGRATOR_BIN" ] || { echo "Could not find signoz-schema-migrator binary"; exit 1; }
		chmod +x "$MIGRATOR_BIN"
		source "${POURS_DIR}/linux/signoz.env"

		# Wait for ClickHouse to be ready
		wait_for_service 'wget --spider -q 0.0.0.0:8123/ping' 'ClickHouse'

		"$MIGRATOR_BIN" sync --dsn="${SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN}" --replication=true --up=
		"$MIGRATOR_BIN" async --dsn="${SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN}" --replication=true --up=
		rm -f /tmp/signoz-schema-migrator.tar.gz
		rm -rf /tmp/signoz-schema-migrator_linux_*

		"""

	_installPostgres: """
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
		        sudo -u "${POSTGRES_USER}" /usr/lib/postgresql/*/bin/initdb -D "$PGDATA" || true
		    else
		        sudo -u "${POSTGRES_USER}" /usr/bin/initdb -D "$PGDATA" || true
		    fi
		fi

		# Copy PostgreSQL configuration files
		mkdir -p "${POSTGRES_CONFIG_DIR}"
		require_pours_file "${POURS_DIR}/postgres/auth.env"
		cp "${POURS_DIR}/postgres/auth.env" "${POSTGRES_CONFIG_DIR}/postgresql.env"

		# Source auth.env to get credentials
		. "${POSTGRES_CONFIG_DIR}/postgresql.env"

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
		echo "PGDATA=$PGDATA" >> "${POSTGRES_CONFIG_DIR}/postgresql.env"

		# Start PostgreSQL to create database and user
		if command_exists systemctl; then
		    systemctl start postgresql || service postgresql start || true
		    sleep 2
		fi

		# Create database and user if they don't exist
		sudo -u "${POSTGRES_USER}" psql -c "SELECT 1 FROM pg_roles WHERE rolname='${POSTGRES_DB_USER:-signoz}'" | grep -q 1 || \\
		    sudo -u "${POSTGRES_USER}" psql -c "CREATE USER ${POSTGRES_DB_USER:-signoz} WITH PASSWORD '${POSTGRES_DB_PASSWORD:-Signoz@123}';" || true

		sudo -u "${POSTGRES_USER}" psql -c "SELECT 1 FROM pg_database WHERE datname='${POSTGRES_DB_NAME:-signoz}'" | grep -q 1 || \\
		    sudo -u "${POSTGRES_USER}" psql -c "CREATE DATABASE ${POSTGRES_DB_NAME:-signoz} OWNER ${POSTGRES_DB_USER:-signoz};" || true

		# Grant privileges
		sudo -u "${POSTGRES_USER}" psql -c "GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB_NAME:-signoz} TO ${POSTGRES_DB_USER:-signoz};" || true

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

		"""

	_installSignoz: """
		# Install SigNoz
		ARCH=$(detect_arch)
		cd /tmp
		curl -L "${SIGNOZ_URL}" -o signoz.tar.gz
		tar -xzf signoz.tar.gz
		mkdir -p "${SIGNOZ_DIR}" "${SIGNOZ_DATA_DIR}" "${SIGNOZ_DIR}/conf"
		cp -r "signoz_linux_${ARCH}"/* "${SIGNOZ_DIR}"
		require_pours_file "${POURS_DIR}/linux/signoz.env"
		cp "${POURS_DIR}/linux/signoz.env" "${SIGNOZ_DIR}/conf/systemd.env"
		if ! getent passwd "${SIGNOZ_USER}" >/dev/null; then
		    useradd --system --home "${SIGNOZ_DIR}" --no-create-home --user-group --shell /sbin/nologin "${SIGNOZ_USER}"
		fi
		chown -R "${SIGNOZ_USER}:${SIGNOZ_USER}" "${SIGNOZ_DATA_DIR}" "${SIGNOZ_DIR}"
		require_pours_file "${POURS_DIR}/linux/signoz.service"
		cp "${POURS_DIR}/linux/signoz.service" /etc/systemd/system/signoz.service
		systemctl daemon-reload
		systemctl start signoz.service
		systemctl enable signoz.service
		rm -f /tmp/signoz.tar.gz
		rm -rf "/tmp/signoz_linux_${ARCH}"

		"""

	_installCollector: """
		# Install SigNoz Otel Collector
		ARCH=$(detect_arch)
		cd /tmp
		curl -L "${COLLECTOR_URL}" -o signoz-otel-collector.tar.gz
		tar -xzf signoz-otel-collector.tar.gz
		mkdir -p "${COLLECTOR_DATA_DIR}" "${COLLECTOR_DIR}" "${COLLECTOR_DIR}/conf"
		cp -r "signoz-otel-collector_linux_${ARCH}"/* "${COLLECTOR_DIR}"
		chown -R "${SIGNOZ_USER}:${SIGNOZ_USER}" "${COLLECTOR_DATA_DIR}" "${COLLECTOR_DIR}"
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
		"""

	// Generate the complete script
	output: strings.Join([
		_header,
		_variables,
		"",
		_helperFunctions,
		_installJava,
		_installZookeeper,
		_installClickhouse,
		_runMigrations,
		_installPostgres,
		_installSignoz,
		_installCollector,
	], "\n")
}