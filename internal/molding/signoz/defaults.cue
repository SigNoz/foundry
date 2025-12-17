package signoz

#BaseConfig: #ConfigSpec & {
	version: {
		banner: {
			enabled: *true | bool
		}
	}

	instrumentation: {
		logs: {
			level: *"info" | #LogLevel
		}

		traces: {
			enabled: *false | bool
			processors: {
				batch: {
					exporter: {
						otlp: {
							endpoint: *"localhost:4317" | string
						}
					}
				}
			}
		}

		metrics: {
			enabled: *true | bool
			readers: {
				pull: {
					exporter: {
						prometheus: {
							host: *"0.0.0.0" | string
							port: *9090 | int
						}
					}
				}
			}
		}
	}

	web: {
		enabled:   *true | bool
		prefix:    *"/" | string
		directory: *"/etc/signoz/web" | string
	}

	cache: {
		provider: *"memory" | #CacheProvider

		memory: {
			ttl:              *60000000000 | int
			cleanup_interval: *"1m" | #Duration
		}

		redis: {
			host:     *"localhost" | string
			port:     *6379 | int
			password: *"" | string
			db:       *0 | int
		}
	}

	sqlstore: {
		provider:       *"sqlite" | #SQLStoreProvider
		max_open_conns: *100 | int

		sqlite: {
			path:         *"/var/lib/signoz/signoz.db" | string
			mode:         *"delete" | string
			busy_timeout: *"10s" | #Duration
		}

		postgres: {
			dsn: *"" | string
		}
	}

	apiserver: {
		timeout: {
			default: *"60s" | #Duration
			max:     *"600s" | #Duration
			excluded_routes: *[
				"/api/v1/logs/tail",
				"/api/v3/logs/livetail",
			] | [...string]
		}
		logging: {
			excluded_routes: *[
				"/api/v1/health",
				"/api/v1/version",
				"/",
			] | [...string]
		}
	}

	querier: {
		cache_ttl:              *"168h" | #Duration
		flux_interval:          *"5m" | #Duration
		max_concurrent_queries: *4 | int
	}

	telemetrystore: {
		max_idle_conns: *50 | int
		max_open_conns: *100 | int
		dial_timeout:   *"5s" | #Duration
		provider:       *"clickhouse" | #TelemetryProvider

		clickhouse: {
			dsn:     *"tcp://${CLICKHOUSE_HOST}:9000" | string
			cluster: *"cluster" | string
			settings: {
				max_execution_time:                      *0 | int
				max_execution_time_leaf:                 *0 | int
				timeout_before_checking_execution_speed: *0 | int
				max_bytes_to_read:                       *0 | int
				max_result_rows:                         *0 | int
				ignore_data_skipping_indices:            *"" | string
				secondary_indices_enable_bulk_filtering: *false | bool
			}
		}
	}

	prometheus: {
		active_query_tracker: {
			enabled:        *true | bool
			path:           *"" | string
			max_concurrent: *20 | int
		}
	}

	alertmanager: {
		provider: *"signoz" | #AlertmanagerProvider

		signoz: {
			poll_interval: *"1m" | #Duration
			external_url:  *"http://localhost:8080" | string

			global: {
				resolve_timeout: *"5m" | #Duration
			}

			route: {
				group_by: *["alertname"] | [...string]
				group_interval:  *"1m" | #Duration
				group_wait:      *"1m" | #Duration
				repeat_interval: *"1h" | #Duration
			}

			alerts: {
				gc_interval: *"30m" | #Duration
			}

			silences: {
				max:                  *0 | int
				max_size_bytes:       *0 | int
				maintenance_interval: *"15m" | #Duration
				retention:            *"120h" | #Duration
			}

			nflog: {
				maintenance_interval: *"15m" | #Duration
				retention:            *"120h" | #Duration
			}
		}
	}

	emailing: {
		enabled: *false | bool

		templates: {
			directory: *"/opt/signoz/conf/templates/email" | string
		}

		smtp: {
			address: *"localhost:25" | string
			from:    *"" | string
			hello:   *"" | string

			headers: *{} | {...}

			auth: {
				username: *"" | string
				password: *"" | string
				secret:   *"" | string
				identity: *"" | string
			}

			tls: {
				enabled:              *false | bool
				insecure_skip_verify: *false | bool
				ca_file_path:         *"" | string
				key_file_path:        *"" | string
				cert_file_path:       *"" | string
			}
		}
	}

	sharder: {
		provider: *"noop" | #SharderProvider

		single: {
			org_id: *"org_id" | string
		}
	}

	analytics: {
		enabled: *false | bool
		segment: {
			key: *"" | string
		}
	}

	statsreporter: {
		enabled:  *true | bool
		interval: *"6h" | #Duration
		collect: {
			identities: *true | bool
		}
	}

	gateway: {
		url: *"http://localhost:8080" | string
	}

	tokenizer: {
		provider: *"jwt" | #TokenizerProvider

		lifetime: {
			idle: *"168h" | #Duration
			max:  *"720h" | #Duration
		}

		rotation: {
			interval: *"30m" | #Duration
			duration: *"60s" | #Duration
		}

		jwt: {
			secret: *"secret" | string
		}

		opaque: {
			gc: {
				interval: *"1h" | #Duration
			}
			token: {
				max_per_user: *5 | int
			}
		}
	}
}

#BaseConfig
