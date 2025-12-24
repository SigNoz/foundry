package clickhouse

_server: {
	clickhouse: {
		// ClickHouse override semantics
		'@replace': *"true" | string
		logger: {
			level: *"information" | #LoggerLevel

			formatting: {
				type: *"json" | #LoggerFormattingType
			}

			log:      *"/var/log/clickhouse-server/clickhouse-server.log" | string
			errorlog: *"/var/log/clickhouse-server/clickhouse-server.err.log" | string
			size:     *"1000M" | string
			count:    *10 | int
		}
		display_name: *"cluster" | string
		listen_host:  *"0.0.0.0" | string

		http_port: *8123 | int
		tcp_port:  *9000 | int
		user_directories: {
			users_xml: {
				path: *"users.xml" | string
			}
			local_directory: {
				path: *"/var/lib/clickhouse/access/" | string
			}
		}
		distributed_ddl: {
			path: *"/clickhouse/task_queue/ddl" | string
		}
		remote_servers: {
			cluster: {
				shard: {
					replica: {
						host: *"${CLICKHOUSE_HOST}" | string
						port: *"${CLICKHOUSE_PORT}" | string | int
					}
				}
			}
		}
		zookeeper: {
			node: {
				host: *"${ZOOKEEPER_HOST}" | string
				port: *"${ZOOKEEPER_PORT}" | string | int
			}
		}
		macros: {
			shard:   *"01" | string
			replica: *"01" | string
		}
		dictionaries_config:                      *"*_dictionary.xml" | string
		user_defined_executable_functions_config: *"*function.xml" | string
		user_scripts_path:                        *"/var/lib/clickhouse/user_scripts/" | string

		distributed_ddl: {
			path: *"/clickhouse/task_queue/ddl" | string
		}

		// Allow all user/variant extensions
		...
	}
}

// users.xml-style config
_users: {
	clickhouse: {
		profiles: {
			// Default profile
			default: {
				max_memory_usage:      *10000000000 | int
				load_balancing:        *"random" | #LoadBalancing
				user_compressed_cache: *0 | int
				log_queries:           *1 | int
			}
		}

		users: {
			// Default user
			default: {
				password: *"" | string
				networks: {
					ip: *"::/0" | string
				}
				profile:                       *"default" | string
				quota:                         *"default" | string
				access_management:             *1 | int
				named_collection_control:      *1 | int
				show_named_collectin:          *1 | int
				show_named_collection_secrets: *1 | int
			}
		}

		quotas: {
			// Default quota
			default: {
				interval: {
					duration:       *3600 | int
					queries:        *0 | int
					errors:         *0 | int
					result_rows:    *0 | int
					read_rows:      *0 | int
					execution_time: *0 | int
				}
			}
		}
	}
	// Allow all user/variant extensions
	...
}

// custom-function.xml style config
_customFunction: {
	functions: {
		function: {
			type:        "executable"
			name:        "histogramQuantile"
			return_type: "Float64"
			argument: [
				{
					type: "Array(Float64)"
					name: "buckets"
				},
				{
					type: "Array(Float64)"
					name: "counts"
				},
				{
					type: "Array(Float64)"
					name: "quantile"
				},
			]
			format:  "CSV"
			command: "./histogramQunatile"
		}
	}
}

#BaseConfig: #ConfigSpec & {
	serverConfig:         _server
	usersConfig:          _users
	customFunctionConfig: _customFunction
}

#BaseConfig
