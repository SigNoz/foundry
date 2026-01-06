package signoz

import (
	"strings"
	"list"
)

// Input parameters - these can be injected from Go
#Params: {
	VERSION:            string | *"v0.101.0"
	OTELCOL_TAG:        string | *"v0.129.9"
	CLICKHOUSE_VERSION: string | *"25.5.6"
	ZOOKEEPER_VERSION:  string | *"3.7.1"
}

// Replica configuration for services that can be scaled
#ReplicaConfig: {
	zookeeper:  int | *1 // Default 3 replicas for zookeeper cluster
	clickhouse: int | *1 // Default 1 replica for clickhouse
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

	// Zookeeper component - can be replicated
	zookeeper: {
		#input: {
			index: int
			total: int
		}

		let _params = params

		#Common
		image:          "signoz/zookeeper:\(_params.ZOOKEEPER_VERSION)"
		container_name: "signoz-zookeeper-\(#input.index)"
		user:           "root"
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
		volumes: ["zookeeper-\(#input.index):/bitnami/zookeeper"]
		environment: {
			ZOO_ENABLE_AUTH: true
			ZOO_SERVER_ID:   "\(#input.index)"
			ZOO_SERVERS: strings.Join([for i in list.Range(1, #input.total+1, 1) {
				"server.\(i)=signoz-zookeeper-\(i):2888:3888"
			}], " ")
		}
	}

	// ClickHouse component - can be replicated
	clickhouse: {
		#input: {
			index: int
			total: int
		}

		let _params = params
		let _replicas = replicas

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
		} & {
			// Add dependencies on all zookeeper instances
			for i in list.Range(1, _replicas.zookeeper+1, 1) {
				"zookeeper-\(i)": condition: "service_healthy"
			}
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

		volumes: [
			"clickhouse-\(#input.index):/var/lib/clickhouse/",
			"./pours/clickhouse/config.xml:/etc/clickhouse-server/config.xml",
			"./pours/clickhouse/users.xml:/etc/clickhouse-server/users.xml",
			"./pours/clickhouse/custom-function.xml:/etc/clickhouse-server/custom-function.xml",
			"./pours/clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/",
		]
		environment: {
			CLICKHOUSE_REPLICA_ID: "\(#input.index)"
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
		volumes: ["./pours/clickhouse/user_scripts:/var/lib/clickhouse/user_scripts/"]
	}

	// SigNoz - singleton service
	signoz: {
		let _params = params
		let _replicas = replicas

		#Common
		image:          "signoz/signoz:\(_params.VERSION)"
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
	}

	// OTel Collector - singleton service
	"otel-collector": {
		let _params = params
		let _replicas = replicas

		#Common
		image:          "signoz/signoz-otel-collector:\(_params.OTELCOL_TAG)"
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
		image:          "signoz/signoz-schema-migrator:\(_params.OTELCOL_TAG)"
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
		image:          "signoz/signoz-schema-migrator:\(_params.OTELCOL_TAG)"
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
		// Generate zookeeper replicas
		for i in list.Range(1, replicas.zookeeper+1, 1) {
			"zookeeper-\(i)": components.zookeeper & {
				#input: {
					index: i
					total: replicas.zookeeper
				}
			}
		}

		// Generate clickhouse replicas
		for i in list.Range(1, replicas.clickhouse+1, 1) {
			"clickhouse-\(i)": components.clickhouse & {
				#input: {
					index: i
					total: replicas.clickhouse
				}
			}
		}

		// Add singleton services
		"init-clickhouse":       components["init-clickhouse"]
		"signoz":                components.signoz
		"otel-collector":        components["otel-collector"]
		"schema-migrator-sync":  components["schema-migrator-sync"]
		"schema-migrator-async": components["schema-migrator-async"]
	}
}

// Generate volumes based on replicas
#GenerateVolumes: {
	replicas: #ReplicaConfig

	volumes: {
		// Zookeeper volumes
		for i in list.Range(1, replicas.zookeeper+1, 1) {
			"zookeeper-\(i)": name: "signoz-zookeeper-\(i)"
		}

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
