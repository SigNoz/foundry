package signoz

// Input parameters - these can be injected from Go
#Params: {
	SIGNOZ_VERSION:     string | *"v0.101.0" // SigNoz version
	OTELCOL_VERSION:    string | *"v0.129.9" // OTel Collector version
	CLICKHOUSE_VERSION: string | *"25.5.6"   // ClickHouse version
	ZOOKEEPER_VERSION:  string | *"3.7.1"    // Zookeeper version
}

// Common configuration template
#Common: {
	networks: ["signoz-net"]
	restart: *"unless-stopped" | string
	logging: {
		options: {
			"max-size": "50m"
			"max-file": "3"
		}
	}
}

// ClickHouse defaults template
#ClickHouseDefaults: {
	params: #Params
	#Common
	image: "clickhouse/clickhouse-server:\(params.CLICKHOUSE_VERSION)"
	tty:   true
	labels: {
		"signoz.io/scrape": "true"
		"signoz.io/port":   "9363"
		"signoz.io/path":   "/metrics"
	}
	depends_on: {
		"init-clickhouse": condition: "service_completed_successfully"
		"zookeeper-1": condition:     "service_healthy"
	}
	healthcheck: {
		test: ["CMD", "wget", "--spider", "-q", "0.0.0.0:8123/ping"]
		interval: "30s"
		timeout:  "5s"
		retries:  3
	}
	ulimits: {
		nproc: 65535
		nofile: {
			soft: 262144
			hard: 262144
		}
	}
	//env_file: "../clickhouse/.env"
}

// Zookeeper defaults template
#ZookeeperDefaults: {
	params: #Params
	#Common
	image: "signoz/zookeeper:\(params.ZOOKEEPER_VERSION)"
	user:  "root"
	labels: {
		"signoz.io/scrape": "true"
		"signoz.io/port":   "9141"
		"signoz.io/path":   "/metrics"
	}
	healthcheck: {
		test: ["CMD-SHELL", "curl -s -m 2 http://localhost:8080/commands/ruok | grep error | grep null"]
		interval: "30s"
		timeout:  "5s"
		retries:  3
	}
}

// DB dependency template
#DbDepend: {
	#Common
	depends_on: {
		clickhouse: condition:             "service_healthy"
		"schema-migrator-sync": condition: "service_completed_successfully"
	}
}

// Main docker-compose structure with parameters
#DockerCompose: {
	params: #Params

	services: {
		"init-clickhouse": {
			#Common
			image:          "clickhouse/clickhouse-server:\(params.CLICKHOUSE_VERSION)"
			container_name: "signoz-init-clickhouse"
			command: ["bash", "-c", """
				version="v0.0.1"
				node_os=$(uname -s | tr '[:upper:]' '[:lower:]')
				node_arch=$(uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)
				echo "Fetching histogram-binary for ${node_os}/${node_arch}"
				cd /tmp
				wget -O histogram-quantile.tar.gz "https://github.com/SigNoz/signoz/releases/download/histogram-quantile%2F${version}/histogram-quantile_${node_os}_${node_arch}.tar.gz"
				tar -xvzf histogram-quantile.tar.gz
				mv histogram-quantile /var/lib/clickhouse/user_scripts/histogramQuantile
				"""]
			restart: "on-failure"
			volumes: ["./pours/clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/"]
		}

		"zookeeper-1": {
			#ZookeeperDefaults
			params:         params
			container_name: "signoz-zookeeper-1"
			volumes: ["zookeeper-1:/bitnami/zookeeper"]
			//env_file: "../zookeeper/.env"
		}

		clickhouse: {
			#ClickHouseDefaults
			params:         params
			container_name: "signoz-clickhouse"
			volumes: [
				"./pours/clickhouse/config.xml:/etc/clickhouse-server/config.xml",
				"./pours/clickhouse/users.xml:/etc/clickhouse-server/users.xml",
				"./pours/clickhouse/custom-function.xml:/etc/clickhouse-server/custom-function.xml",
				"./pours/clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/",
				"clickhouse:/var/lib/clickhouse/",
			]
		}

		signoz: {
			#DbDepend
			image:          "signoz/signoz:\(params.SIGNOZ_VERSION)"
			container_name: "signoz"
			ports: ["8080:8080"]
			volumes: ["sqlite:/var/lib/signoz/"]
			//env_file: "../signoz/.env"
			healthcheck: {
				test: ["CMD", "wget", "--spider", "-q", "localhost:8080/api/v1/health"]
				interval: "30s"
				timeout:  "5s"
				retries:  3
			}
		}

		"otel-collector": {
			#DbDepend
			image:          "signoz/signoz-otel-collector:\(params.OTELCOL_VERSION)"
			container_name: "signoz-otel-collector"
			command: [
				"--config=/etc/otel-collector-config.yaml",
				"--manager-config=/etc/manager-config.yaml",
				"--copy-path=/var/tmp/collector-config.yaml",
				"--feature-gates=-pkg.translator.prometheus.NormalizeName",
			]
			volumes: [
				"./pours/otel-collector/otel-collector-config.yaml:/etc/otel-collector-config.yaml",
				"./pours/otel-collector/otel-collector-opamp-config.yaml:/etc/manager-config.yaml",
			]
			//env_file: "../otel-collector/.env"
			ports: ["4317:4317", "4318:4318"]
			depends_on: {
				clickhouse: condition:             "service_healthy"
				"schema-migrator-sync": condition: "service_completed_successfully"
				signoz: condition:                 "service_healthy"
			}
		}

		"schema-migrator-sync": {
			#Common
			image:          "signoz/signoz-schema-migrator:\(params.OTELCOL_VERSION)"
			container_name: "schema-migrator-sync"
			command: ["sync", "--dsn=tcp://clickhouse:9000", "--up="]
			depends_on: clickhouse: condition: "service_healthy"
			restart: "on-failure"
		}

		"schema-migrator-async": {
			#DbDepend
			image:          "signoz/signoz-schema-migrator:\(params.OTELCOL_VERSION)"
			container_name: "schema-migrator-async"
			command: ["async", "--dsn=tcp://clickhouse:9000", "--up="]
			restart: "on-failure"
		}
	}

	networks: "signoz-net": name: "signoz-net"

	volumes: {
		clickhouse: name:    "signoz-clickhouse"
		sqlite: name:        "signoz-sqlite"
		"zookeeper-1": name: "signoz-zookeeper-1"
	}
}

// Default instance with default parameters
compose: #DockerCompose & {
	params: #Params
}
