package signoz

import (
	"list"
)

// Input parameters - these can be injected from Go
#Params: {
	SIGNOZ_VERSION:     string | *"v0.101.0"
	OTELCOL_VERSION:    string | *"v0.129.9"
	CLICKHOUSE_VERSION: string | *"25.5.6"
}

// Replica configuration for services that can be scaled
#ReplicaConfig: {
	clickhouse:    int | *1 // Default 1 replica for clickhouse
	otelcollector: int | *1 // Default 1 replica for otel-collector
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

// Component definitions - these are the building blocks
#Components: {
	params:   #Params
	replicas: #ReplicaConfig

	// ClickHouse component - can be replicated
	clickhouse: {
		#input: {
			index: int
			total: int
		}

		let _params = params

		#Common
		image:          "clickhouse/clickhouse-server:\(_params.CLICKHOUSE_VERSION)"
		container_name: "signoz-clickhouse-\(#input.index)"
		tty:            true
		labels: {
			"signoz.io/scrape": "true"
			"signoz.io/port":   "9363"
			"signoz.io/path":   "/metrics"
		}
		depends_on: {
			"init-clickhouse": condition: "service_completed_successfully"
		}
		entrypoint: *[
			"/usr/bin/clickhouse-server",
			"--config-file=/etc/clickhouse-server/config.yaml",
		] | [...string]
		healthcheck: {
			test: ["CMD", "wget", "--spider", "-q", "0.0.0.0:8123/ping"]
			interval: "30s"
			timeout:  "5s"
			retries:  3
		}
		user: "clickhouse:clickhouse"
		ulimits: {
			nproc: 65535
			nofile: {
				soft: 262144
				hard: 262144
			}
		}
		volumes: [
			"clickhouse-\(#input.index):/var/lib/clickhouse/",
			"../clickhouse/config.yaml:/etc/clickhouse-server/config.yaml",
			"../clickhouse/custom-function.yaml:/etc/clickhouse-server/custom-function.yaml",
			"../clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/",
		]
		environment: {
			CLICKHOUSE_REPLICA_ID:      "\(#input.index)"
			CLICKHOUSE_SKIP_USER_SETUP: 1
		}
	}

	// Init ClickHouse - singleton service
	"init-clickhouse": {
		let _params = params

		#Common
		image:          "clickhouse/clickhouse-server:\(_params.CLICKHOUSE_VERSION)"
		container_name: "signoz-init-clickhouse"
		command: ["bash", "-c", #"""
			version="v0.0.1"
			node_os=$$(uname -s | tr '[:upper:]' '[:lower:]')
			node_arch=$$(uname -m | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)
			echo "Fetching histogram-binary for $${node_os}/$${node_arch}"
			cd /tmp
			wget -O histogram-quantile.tar.gz "https://github.com/SigNoz/signoz/releases/download/histogram-quantile%2F$${version}/histogram-quantile_$${node_os}_$${node_arch}.tar.gz"
			tar -xvzf histogram-quantile.tar.gz
			mv histogram-quantile /var/lib/clickhouse/user_scripts/histogramQuantile
			"""#]
		restart: "on-failure"
		volumes: ["../clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/"]
	}

	// SigNoz - singleton service
	signoz: {
		let _params = params
		let _replicas = replicas

		#Common
		image:          "signoz/signoz:v\(_params.SIGNOZ_VERSION)"
		container_name: "signoz"
		ports: ["8080:8080"]
		volumes: ["sqlite:/var/lib/signoz/"]
		depends_on: {
			"schema-migrator-sync": condition: "service_completed_successfully"
		} & {
			// Depend on all clickhouse instances
			for i in list.Range(1, _replicas.clickhouse+1, 1) {
				"clickhouse-\(i)": condition: "service_healthy"
			}
		}
		healthcheck: {
			test: ["CMD", "wget", "--spider", "-q", "localhost:8080/api/v1/health"]
			interval: "30s"
			timeout:  "5s"
			retries:  3
		}
		environment: {
			SIGNOZ_ALERTMANAGER_PROVIDER:         "signoz"
			SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN: "tcp://clickhouse-1:9000"
			SIGNOZ_SQLSTORE_SQLITE_PATH:          "/var/lib/signoz/signoz.db"
			DASHBOARDS_PATH:                      "/root/config/dashboards"
			STORAGE:                              "clickhouse"
			GODEBUG:                              "netdns:go"
			TELEMETRY_ENABLED:                    true
			DEPLOYMENT_TYPE:                      "docker-swarm"
			DOT_METRICS_ENABLED:                  true
		}
	}

	// OTel Collector - singleton service
	otelcollector: {
		let _params = params
		let _replicas = replicas

		#Common
		image:          "signoz/signoz-otel-collector:v\(_params.OTELCOL_VERSION)"
		container_name: "signoz-otel-collector"
		command: [
			"--config=/etc/otel-collector-config.yaml",
			"--manager-config=/etc/manager-config.yaml",
			"--copy-path=/var/tmp/collector-config.yaml",
			"--feature-gates=-pkg.translator.prometheus.NormalizeName",
		]
		volumes: [
			"../signozOtelCollectorConfig/config.yaml:/etc/otel-collector-config.yaml",
			"../signozOtelCollectorConfig/config.yaml:/etc/manager-config.yaml",
		]
		ports: ["4317:4317", "4318:4318"]
		depends_on: {
			"schema-migrator-sync": condition: "service_completed_successfully"
			signoz: condition:                 "service_healthy"
		} & {
			// Depend on all clickhouse instances
			for i in list.Range(1, _replicas.clickhouse+1, 1) {
				"clickhouse-\(i)": condition: "service_healthy"
			}
		}
	}

	// Schema Migrator Sync - singleton service
	"schema-migrator-sync": {
		let _params = params

		#Common
		image:          "signoz/signoz-schema-migrator:v\(_params.OTELCOL_VERSION)"
		container_name: "schema-migrator-sync"
		command: ["sync", "--dsn=tcp://clickhouse-1:9000", "--up="]
		depends_on: {
			"clickhouse-1": condition: "service_healthy"
		}
		restart: "on-failure"
	}

	// Schema Migrator Async - singleton service
	"schema-migrator-async": {
		let _params = params
		let _replicas = replicas

		#Common
		image:          "signoz/signoz-schema-migrator:v\(_params.OTELCOL_VERSION)"
		container_name: "schema-migrator-async"
		command: ["async", "--dsn=tcp://clickhouse-1:9000", "--up="]
		depends_on: {
			"schema-migrator-sync": condition: "service_completed_successfully"
		} & {
			// Depend on all clickhouse instances
			for i in list.Range(1, _replicas.clickhouse+1, 1) {
				"clickhouse-\(i)": condition: "service_healthy"
			}
		}
		restart: "on-failure"
	}
}

// Generate services by expanding replicated components
#GenerateServices: {
	params:   #Params
	replicas: #ReplicaConfig

	let components = #Components & {
		"params":   params
		"replicas": replicas
	}

	services: {
		// Generate clickhouse replicas
		for i in list.Range(1, replicas.clickhouse+1, 1) {
			"clickhouse-\(i)": components.clickhouse & {
				#input: {
					index: i
					total: replicas.clickhouse
				}
			}
		}

		// Generate otel-collector replicas
		for i in list.Range(1, replicas.otelcollector+1, 1) {
			"otelcollector-\(i)": components.otelcollector & {
				#input: {
					index: i
					total: replicas.otelcollector
				}
			}
		}

		// Add singleton services
		"init-clickhouse":       components["init-clickhouse"]
		"signoz":                components.signoz
		"schema-migrator-sync":  components["schema-migrator-sync"]
		"schema-migrator-async": components["schema-migrator-async"]
	}
}

// Generate volumes based on replicas
#GenerateVolumes: {
	replicas: #ReplicaConfig

	volumes: {
		// ClickHouse volumes
		for i in list.Range(1, replicas.clickhouse+1, 1) {
			"clickhouse-\(i)": name: "signoz-clickhouse-\(i)"
		}

		// Singleton volumes
		sqlite: name: "signoz-sqlite"
	}
}

// Main docker-compose structure
#DockerCompose: {
	params:   #Params
	replicas: #ReplicaConfig

	version: "3"

	let generated = #GenerateServices & {
		"params":   params
		"replicas": replicas
	}

	let generatedVolumes = #GenerateVolumes & {
		"replicas": replicas
	}

	services: generated.services
	volumes:  generatedVolumes.volumes

	networks: "signoz-net": name: "signoz-net"
}

// Default instance with default parameters
compose: #DockerCompose & {
	params:   #Params
	replicas: #ReplicaConfig
}
