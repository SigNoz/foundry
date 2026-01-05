package linux

import "strings"

// Linux-specific configuration overrides
#LinuxOverrides: {
	_#inputs: {
		clickhouseHost: string
		clickhousePort: string | int
	}
	// ClickHouse configuration for Linux platform
	out:
	{
		clickhouse: {
		config: {
			serverConfig: {
				clickhouse: {
					remote_servers: {
						cluster: {
							shard: {
								replica: {
									host: _#inputs.clickhouseHost  // Fixed: added underscore
									port: _#inputs.clickhousePort  // Fixed: added underscore
								}
							}
						}
					}
					zookeeper: {
						node: {
							host: "127.0.0.1"
							port: "2181"
						}
					}
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
		// Restart behavior with defaults
		Restart:    string & =~"^(no|on-success|on-failure|on-abnormal|on-watchdog|on-abort|always)$" | *"on-failure"
		TimeoutSec: *30 | int & >=0
		
		// Resource limits
		LimitNOFILE?:  int
		LimitNPROC?:   int
		MemoryLimit?:  string
		CPUQuota?:     string
		
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
		WorkingDirectory: string | *"${ZOOKEEPER_INSTALL_DIR}"
		EnvironmentFile: string | *"${ZOOKEEPER_INSTALL_DIR}/conf/zoo.env"
		ExecStart: "${ZOOKEEPER_INSTALL_DIR}/bin/zkServer.sh start ${ZOOKEEPER_INSTALL_DIR}/conf/zoo.cfg"
		ExecStop: "${ZOOKEEPER_INSTALL_DIR}/bin/zkServer.sh stop ${ZOOKEEPER_INSTALL_DIR}/conf/zoo.cfg"
		ExecReload: "${ZOOKEEPER_INSTALL_DIR}/bin/zkServer.sh restart ${ZOOKEEPER_INSTALL_DIR}/conf/zoo.cfg"
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