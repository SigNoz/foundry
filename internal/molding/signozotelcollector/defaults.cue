package signozotelcollector

_clickhouse: {
	host: *"clickhouse" | string @tag(CLICKHOUSE_HOST)
	port: *9000 | int            @tag(CLICKHOUSE_PORT)
}

#BaseConfig: #ConfigSpec & {
	connectors: {
		signozmeter: {
			metrics_flush_interval: *"1h" | string
			dimensions: *[
				{name: "service.name"},
				{name: "deployment.environment"},
				{name: "host.name"},
			] | [...{name: string}]
		}
	}
	receivers: {
		otlp: {
			protocols: {
				grpc: {endpoint: *"0.0.0.0:4317" | string}
				http: {endpoint: *"0.0.0.0:4318" | string}
			}
		}

		prometheus: {
			config: {
				global: {
					scrape_interval: *"60s" | string
				}
				scrape_configs: *[{
					job_name: "otel-collector"
					static_configs: [{
						targets: ["localhost:8888"]
						labels: {
							job_name: "otel-collector"
						}
					}]
				}] | [...#ScrapeConfig]
			}
		}
	}

	processors: {
		batch: {
			send_batch_size:     *10000 | int
			send_batch_max_size: *11000 | int
			timeout:             *"10s" | string
		}
		"batch/meter": {
			send_batch_size:     *20000 | int
			send_batch_max_size: *25000 | int
			timeout:             *"1s" | string
		}
		resourcedetection: {
			detectors: *["env", "system"] | [...string]
			timeout: *"2s" | string
		}

		"signozspanmetrics/delta": {
			metrics_exporter:       *"signozclickhousemetrics" | string
			metrics_flush_interval: *"60s" | string
			latency_histogram_buckets: *[
				"100us", "1ms", "2ms", "6ms", "10ms", "50ms", "100ms",
				"250ms", "500ms", "1000ms", "1400ms", "2000ms",
				"5s", "10s", "20s", "40s", "60s",
			] | [...string]
			dimensions_cache_size:   *100000 | int
			aggregation_temporality: *"AGGREGATION_TEMPORALITY_DELTA" | string
			enable_exp_histogram:    *true | bool
			dimensions: *[
				{name: "service.namespace", default: "default"},
				{name: "deployment.environment", default: "default"},
				{name: "signoz.collector.id"},
				{name: "service.version"},
				{name: "browser.platform"},
				{name: "browser.mobile"},
				{name: "k8s.cluster.name"},
				{name: "k8s.node.name"},
				{name: "k8s.namespace.name"},
				{name: "host.name"},
				{name: "host.type"},
				{name: "container.name"},
			] | [...{name: string, default?: string}]
		}
	}

	extensions: {
		health_check: {endpoint: *"0.0.0.0:13133" | string}
		pprof: {endpoint: *"0.0.0.0:1777" | string}
	}

	exporters: {
		clickhousetraces: {
			datasource:                      *"tcp://\(_clickhouse.host):\(_clickhouse.port)/signoz_traces" | string
			low_cardinal_exception_grouping: *"${env:LOW_CARDINAL_EXCEPTION_GROUPING}" | string
			use_new_schema:                  *true | bool
		}
		signozclickhousemetrics: {
			dsn: *"tcp://\(_clickhouse.host):\(_clickhouse.port)/signoz_metrics" | string
		}
		clickhouselogsexporter: {
			dsn:            *"tcp://\(_clickhouse.host):\(_clickhouse.port)/signoz_logs" | string
			timeout:        *"10s" | string
			use_new_schema: *true | bool
		}
		signozclickhousemeter: {
			dsn:     *"tcp://\(_clickhouse.host):\(_clickhouse.port)/signoz_meter" | string
			timeout: *"45s" | string
			sending_queue:
				enabled: *false | bool
		}
	}

	service: {
		telemetry: logs: {encoding: *"json" | string}

		extensions: *["health_check", "pprof"] | [...string]

		pipelines: {
			traces: {
				receivers: *["otlp"] | [...string]
				processors: *["signozspanmetrics/delta", "batch"] | [...string]
				exporters: *["clickhousetraces", "signozmeter"] | [...string]
			}

			metrics: {
				receivers: *["otlp"] | [...string]
				processors: *["batch"] | [...string]
				exporters: *["signozclickhousemetrics", "signozmeter"] | [...string]
			}

			"metrics/prometheus": {
				receivers: *["prometheus"] | [...string]
				processors: *["batch"] | [...string]
				exporters: *["signozclickhousemetrics", "signozmeter"] | [...string]
			}

			logs: {
				receivers: *["otlp"] | [...string]
				processors: *["batch"] | [...string]
				exporters: *["clickhouselogsexporter", "signozmeter"] | [...string]
			}

			"metrics/meter": {
				receivers: *["signozmeter"] | [...string]
				processors: *["batch/meter"] | [...string]
				exporters: *["signozclickhousemeter"] | [...string]
			}
		}
	}
}
#BaseConfig
