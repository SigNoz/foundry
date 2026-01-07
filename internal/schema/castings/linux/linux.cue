package linux

import (
	"strings"
	"list"
	"strconv"
)

// Linux-specific configuration overrides
#Overrides: {
	inputs: {
		clickhouse: {
			host:     string | *"127.0.0.1"
			port:     int | string | *9000
			replicas: int | *1
		}
		zookeeper: {
			host: string | *"127.0.0.1"
			port: int | string | *2181
			replicas: int | *1
		}

		postgres: {
			host: string | *"127.0.0.1"
			port: int | string | *5432
			auth: {
				postgres_password: string & =~"[a-z]+" & =~"[A-Z]+" & =~"[0-9]+" & =~"[!@#$%^&*]+"
				postgres_db:       *"signoz" | string
				postgres_user:     *"signoz" | string
			}
		}
	}

	// Helper to normalize port to int (handles both int and string input)
	_portToInt: {
		clickhouse: (inputs.clickhouse.port & int) | strconv.Atoi(inputs.clickhouse.port & string)
		postgres:   (inputs.postgres.port & int) | strconv.Atoi(inputs.postgres.port & string)
		zookeeper:  (inputs.zookeeper.port & int) | strconv.Atoi(inputs.zookeeper.port & string)
	}

	// Helper to convert port to string
	_portToString: {
		clickhouse: "\(inputs.clickhouse.port)"
		postgres:   "\(inputs.postgres.port)"
		zookeeper:  "\(inputs.zookeeper.port)"
	}
	
	// ClickHouse configuration for Linux platform
	out:
	{
		zookeeper:{
			config:{
				"admin.serverPort": 3181
			}
		}
		clickhouse: {
		config: {
			serverConfig: {
					remote_servers: {
						cluster: {
							shard: [
								 {
									replica: [
										for i, _ in list.Range(0, inputs.clickhouse.replicas, 1)
										{
										host: inputs.clickhouse.host
										port: _portToInt.clickhouse + i // 9000, 9001, 9002, ...
									}]
								},
							]
						}
					}
					zookeeper: {
						node: [
							for i, _ in list.Range(0, inputs.zookeeper.replicas, 1){
							host: inputs.zookeeper.host
							port: _portToInt.zookeeper + i
						}]
					}
				
			}
		}
		}

		signoz: {
			config:{
				telemetrystore: {
					clickhouse: {
						dsn: "tcp://" + inputs.clickhouse.host + ":" + _portToString.clickhouse
					}
				}
				
				sqlstore: {
					provider: "postgres"
					postgres: {
						dsn: "postgres://" + inputs.postgres.auth.postgres_user + ":" + inputs.postgres.auth.postgres_password + "@" + inputs.postgres.host + ":" + _portToString.postgres + "/" + inputs.postgres.auth.postgres_db
					}
				}
		}
		}

		signozOtelCollector: {
			config: {
				exporters:{
					clickhousetraces:{
						datasource: "tcp://" + inputs.clickhouse.host + ":" + _portToString.clickhouse + "/signoz_traces"
					}
					signozclickhousemetrics:{
						dsn: "tcp://" + inputs.clickhouse.host + ":" + _portToString.clickhouse + "/signoz_metrics"
					}
					clickhouselogsexporter:{
						dsn: "tcp://" + inputs.clickhouse.host + ":" + _portToString.clickhouse + "/signoz_logs"
					}
					signozclickhousemeter:{
						dsn: "tcp://" + inputs.clickhouse.host + ":" + _portToString.clickhouse + "/signoz_meter"
					}
				}
			}
	}
	
	
}
}

// SystemdUnit definition
#Deployment: {
	// Unit section parameters
	unit: {
		Description:   string & !="" | *"Service"
		After?: 		string
		Requires?:     string
		Wants?:        string
		Documentation?:string
		[string]:      string  // Allow additional fields
		
	}
	
	// Service section with defaults
	service: {
		Type:             string & =~"^(simple|forking|oneshot|dbus|notify|idle)$" | *"simple"
		User?:            string
		Group?:           string
		WorkingDirectory?: string
		EnvironmentFile?: string
		
		// Execution commands
		ExecStart:   string & !=""
		ExecStop?:   string
		ExecReload?: string
		KillMode?: string
		KillSignal?: string
		
		// Restart behavior with defaults
		Restart:    string & =~"^(no|on-success|on-failure|on-abnormal|on-watchdog|on-abort|always)$" | *"on-failure"
		RestartSec?: int & >=0
		TimeoutSec?: int & >=0
		TimeoutStopSec?: string | int
		TimeoutStartSec?: int & >=0
		
		// Runtime directories
		RuntimeDirectory?: string
		RuntimeDirectoryMode?: string
		
		// Resource limits
		LimitNOFILE?:  int
		LimitNPROC?:   int
		LimitCORE?:    string | int
		MemoryLimit?:  string
		CPUQuota?:     string
		
		// Capabilities
		CapabilityBoundingSet?: string
		AmbientCapabilities?: string
		
		[string]: string | int  // Allow additional fields
	}
	
	// Install section
	install: {
		WantedBy:  string | *"multi-user.target"
		[string]:   string  // Allow additional fields
	}
	
	// Generate the output
	output: strings.Join([
		_unitSection,
		"",
		_serviceSection,
		"",
		_installSection,
	], "\n")
	
	// Private helper to generate Unit section
	_unitSection: strings.Join([
		"[Unit]",
		for k, v in unit {
			"\(k)=\(v)"
		},
	], "\n")
	
	// helper to generate Service section
	_serviceSection: strings.Join([
		"[Service]",
		for k, v in service {
				// Handle string values
				"\(k)=\(v)"
		},
	], "\n")
	
	// helper to generate Install section
	_installSection: strings.Join([
		"[Install]",
		for k, v in install {
			"\(k)=\(v)"
		},
	], "\n")
}

// Zookeeper service with defaults applied
#ZookeeperService: #Deployment & {
	unit: {
		Description: "Apache Zookeeper"
		Documentation: "http://zookeeper.apache.org"
		After: "network.target"
	}
	
	service: {
		Type: "forking"
		User: string | *"zookeeper"
		Group: string | *"zookeeper"
		WorkingDirectory: string | *"/opt/zookeeper"
		ExecStart: "/opt/zookeeper/bin/zkServer.sh start /opt/zookeeper/conf/zoo.cfg"
		ExecStop: "/opt/zookeeper/bin/zkServer.sh stop /opt/zookeeper/conf/zoo.cfg"
		ExecReload: "/opt/zookeeper/bin/zkServer.sh restart /opt/zookeeper/conf/zoo.cfg"
	}
	
	install: {
		// WantedBy uses default: "multi-user.target"
	}
}

// SigNoz service definition
#SignozService: #Deployment & {
	unit: {
		Description: "SigNoz"
		Documentation: "https://signoz.io/docs"
		After: "clickhouse-server.service"
	}
	
	service: {
		Type: "simple"
		User: "signoz"
		Group: "signoz"
		KillMode: "mixed"
		Restart: "on-failure"
		WorkingDirectory: "/opt/signoz"
		EnvironmentFile: "/opt/signoz/conf/systemd.env"
		ExecStart: "/opt/signoz/bin/signoz server"
	}
	
	install: {
		// WantedBy uses default: "multi-user.target"
	}
}

// SigNoz OTel Collector service definition
#SignozOtelCollectorService: #Deployment & {
	unit: {
		Description: "SigNoz OTel Collector"
		Documentation: "https://signoz.io/docs"
		After: "clickhouse-server.service"
	}
	
	service: {
		Type: "simple"
		User: "signoz"
		Group: "signoz"
		KillMode: "mixed"
		Restart: "on-failure"
		WorkingDirectory: "/opt/signoz-otel-collector"
		ExecStart: "/opt/signoz-otel-collector/bin/signoz-otel-collector --config=/opt/signoz-otel-collector/conf/config.yaml --manager-config=/opt/signoz-otel-collector/conf/opamp.yaml --copy-path=/var/lib/signoz-otel-collector/config.yaml"
	}
	
	install: {
		// WantedBy uses default: "multi-user.target"
	}
}


// PostgreSQL service definition
#PostgresService: #Deployment & {
	unit: {
		Description: "PostgreSQL database server"
		Documentation: "https://www.postgresql.org/docs/"
		After: "network.target"
	}
	
	service: {
		Type: "notify"
		User: "postgres"
		Group: "postgres"
		EnvironmentFile: "/etc/postgresql/postgresql.env"
		ExecStart: "/usr/lib/postgresql/bin/postgres -D ${PGDATA}"
		ExecReload: "/bin/kill -HUP $MAINPID"
		KillMode: "mixed"
		KillSignal: "SIGINT"
		TimeoutSec: 0
		Restart: "on-failure"
		LimitNOFILE: 65536
	}
	
	install: {
		// WantedBy uses default: "multi-user.target"
	}
}

// ClickHouse service definition
#ClickHouseService: #Deployment & {
	unit: {
		Description: "ClickHouse Server"
		Requires: "network-online.target"
		After: "time-sync.target network-online.target"
		Wants: "time-sync.target"
	}
	
	service: {
		Type: "simple"
		User: "clickhouse"
		Group: "clickhouse"
		Restart: "always"
		RestartSec: 30
		TimeoutStopSec: "infinity"
		TimeoutStartSec: 0
		RuntimeDirectory: "clickhouse"
		RuntimeDirectoryMode: "0755"
		ExecStart: "/usr/bin/clickhouse-server --config=/etc/clickhouse-server/config.yaml --pid-file=/run/clickhouse/clickhouse.pid"
		LimitCORE: "infinity"
		LimitNOFILE: 500000
		CapabilityBoundingSet: "CAP_NET_ADMIN CAP_IPC_LOCK CAP_SYS_NICE CAP_NET_BIND_SERVICE"
		AmbientCapabilities: "CAP_NET_ADMIN CAP_IPC_LOCK CAP_SYS_NICE CAP_NET_BIND_SERVICE"
	}
	
	install: {
		// WantedBy uses default: "multi-user.target"
	}
}