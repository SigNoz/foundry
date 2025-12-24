package signoz

// Common enums
#LogLevel: "debug" | "info" | "warn" | "error"

#CacheProvider:        "memory" | "redis"
#SQLStoreProvider:     "sqlite" | "postgres" | string
#TelemetryProvider:    "clickhouse" | string
#AlertmanagerProvider: "signoz" | string
#SharderProvider:      "noop" | "single" | string
#TokenizerProvider:    "jwt" | "opaque" | string

// Go-style duration strings: 1s, 5m, 6h, 120h, etc.
#Duration: =~"^[0-9]+(ns|us|µs|ms|s|m|h)$"

#ConfigSpec: {
	version: {
		banner: {
			enabled: bool
		}
	}

	instrumentation: {
		logs: {
			level: #LogLevel
		}

		traces: {
			enabled: bool
			processors: {
				batch: {
					exporter: {
						otlp: {
							endpoint: string
						}
					}
				}
			}
		}

		metrics: {
			enabled: bool
			readers: {
				pull: {
					exporter: {
						prometheus: {
							host: string
							port: int
						}
					}
				}
			}
		}
	}

	web: {
		enabled:   bool
		prefix:    string
		directory: string
	}

	cache: {
		provider: #CacheProvider

		memory: {
			// ns
			ttl:              int
			cleanup_interval: #Duration
		}

		redis: {
			host:     string
			port:     int
			password: string | *""
			db:       int
		}
	}

	sqlstore: {
		provider:       #SQLStoreProvider
		max_open_conns: int

		sqlite: {
			path:         string
			mode:         string
			busy_timeout: #Duration
		}

		postgres: {
			dsn: string
		}
	}

	apiserver: {
		timeout: {
			default: #Duration
			max:     #Duration
			excluded_routes: [...string]
		}
		logging: {
			excluded_routes: [...string]
		}
	}

	querier: {
		cache_ttl:              #Duration
		flux_interval:          #Duration
		max_concurrent_queries: int
	}

	telemetrystore: {
		max_idle_conns: int
		max_open_conns: int
		dial_timeout:   #Duration
		provider:       #TelemetryProvider

		clickhouse: {
			dsn:     string
			cluster: string
			settings: {
				max_execution_time:                      int
				max_execution_time_leaf:                 int
				timeout_before_checking_execution_speed: int
				max_bytes_to_read:                       int
				max_result_rows:                         int
				ignore_data_skipping_indices:            string
				secondary_indices_enable_bulk_filtering: bool
			}
		}
	}

	prometheus: {
		active_query_tracker: {
			enabled:        bool
			path:           string
			max_concurrent: int
		}
	}

	alertmanager: {
		provider: #AlertmanagerProvider

		signoz: {
			poll_interval: #Duration
			external_url:  string

			global: {
				resolve_timeout: #Duration
			}

			route: {
				group_by: [...string]
				group_interval:  #Duration
				group_wait:      #Duration
				repeat_interval: #Duration
			}

			alerts: {
				gc_interval: #Duration
			}

			silences: {
				max:                  int
				max_size_bytes:       int
				maintenance_interval: #Duration
				retention:            #Duration
			}

			nflog: {
				maintenance_interval: #Duration
				retention:            #Duration
			}
		}
	}

	emailing: {
		enabled: bool

		templates: {
			directory: string
		}

		smtp: {
			address: string
			from:    string | *""
			hello:   string | *""

			headers: {...}

			auth: {
				username: string | *""
				password: string | *""
				secret:   string | *""
				identity: string | *""
			}

			tls: {
				enabled:              bool
				insecure_skip_verify: bool
				ca_file_path:         string | *""
				key_file_path:        string | *""
				cert_file_path:       string | *""
			}
		}
	}

	sharder: {
		provider: #SharderProvider

		single: {
			org_id: string
		}
	}

	analytics: {
		enabled: bool
		segment: {
			key: string
		}
	}

	statsreporter: {
		enabled:  bool
		interval: #Duration
		collect: {
			identities: bool
		}
	}

	gateway: {
		url: string
	}

	tokenizer: {
		provider: #TokenizerProvider

		lifetime: {
			idle: #Duration
			max:  #Duration
		}

		rotation: {
			interval: #Duration
			duration: #Duration
		}

		jwt: {
			secret: string
		}

		opaque: {
			gc: {
				interval: #Duration
			}
			token: {
				max_per_user: int
			}
		}
	}
}
