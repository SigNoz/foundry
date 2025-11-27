package otel_collector

// Base configuration schema with defaults
#Config: {
	clickhouse: {
		host:    string @tag(clickhouse_host)
		port:    int | *9000
		timeout: string | *"10s"
	}

	otlp: {
		grpc_endpoint: string | *"0.0.0.0:4317"
		http_endpoint: string | *"0.0.0.0:4318"
	}

	prometheus: {
		scrape_interval: string | *"60s"
		scrape_targets: [...string] | *["localhost:8888"]
	}

	batch: {
		send_size:  int | *10000
		max_size:   int | *11000
		timeout:    string | *"10s"
	}

	resource_detection: {
		detectors: [...string] | *["env", "system"]
		timeout:   string | *"2s"
	}

	span_metrics: {
		exporter:           string | *"signozclickhousemetrics"
		flush_interval:     string | *"60s"
		cache_size:         int | *100000
		temporality:        string | *"AGGREGATION_TEMPORALITY_DELTA"
		enable_exp_histogram: bool | *true
		latency_buckets: [...string] | *[
			"100us", "1ms", "2ms", "6ms", "10ms", "50ms",
			"100ms", "250ms", "500ms", "1000ms", "1400ms",
			"2000ms", "5s", "10s", "20s", "40s", "60s",
		]
		dimensions: [...{
			name:     string
			default?: string
		}] | *[
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
		]
	}

	extensions: {
		health_check_endpoint: string | *"0.0.0.0:13133"
		pprof_endpoint:        string | *"0.0.0.0:1777"
	}

	features: {
		low_cardinal_exception_grouping: bool | *false
		use_new_schema:                 bool | *true
	}

	telemetry: {
		log_encoding: string | *"json"
	}
}

// Receivers builder
#Receivers: {
	_config: #Config

	otlp: {
		protocols: {
			grpc: endpoint: _config.otlp.grpc_endpoint
			http: endpoint: _config.otlp.http_endpoint
		}
	}

	prometheus: {
		config: {
			global: scrape_interval: _config.prometheus.scrape_interval
			scrape_configs: [{
				job_name: "otel-collector"
				static_configs: [{
					targets: _config.prometheus.scrape_targets
					labels: job_name: "otel-collector"
				}]
			}]
		}
	}
}

// Processors builder
#Processors: {
	_config: #Config

	batch: {
		send_batch_size:     _config.batch.send_size
		send_batch_max_size: _config.batch.max_size
		timeout:             _config.batch.timeout
	}

	resourcedetection: {
		detectors: _config.resource_detection.detectors
		timeout:   _config.resource_detection.timeout
	}

	"signozspanmetrics/delta": {
		metrics_exporter:          _config.span_metrics.exporter
		metrics_flush_interval:    _config.span_metrics.flush_interval
		latency_histogram_buckets: _config.span_metrics.latency_buckets
		dimensions_cache_size:     _config.span_metrics.cache_size
		aggregation_temporality:   _config.span_metrics.temporality
		enable_exp_histogram:      _config.span_metrics.enable_exp_histogram
		dimensions:                _config.span_metrics.dimensions
	}
}

// Extensions builder
#Extensions: {
	_config: #Config

	health_check: endpoint: _config.extensions.health_check_endpoint
	pprof: endpoint:        _config.extensions.pprof_endpoint
}

// Exporters builder
#Exporters: {
	_config: #Config
	
	let ch_host = _config.clickhouse.host
	let ch_port = _config.clickhouse.port

	clickhousetraces: {
		datasource: "tcp://\(ch_host):\(ch_port)/signoz_traces"
		low_cardinal_exception_grouping: _config.features.low_cardinal_exception_grouping
		use_new_schema:                 _config.features.use_new_schema
	}

	signozclickhousemetrics: {
		dsn: "tcp://\(ch_host):\(ch_port)/signoz_metrics"
	}

	clickhouselogsexporter: {
		dsn:            "tcp://\(ch_host):/signoz_logs"
		timeout:        _config.clickhouse.timeout
		use_new_schema: _config.features.use_new_schema
	}
}

// Service builder
#Service: {
	_config: #Config

	telemetry: logs: encoding: _config.telemetry.log_encoding

	extensions: [
		"health_check",
		"pprof",
	]

	pipelines: {
		traces: {
			receivers: ["otlp"]
			processors: ["signozspanmetrics/delta", "batch"]
			exporters: ["clickhousetraces"]
		}
		metrics: {
			receivers: ["otlp"]
			processors: ["batch"]
			exporters: ["signozclickhousemetrics"]
		}
		"metrics/prometheus": {
			receivers: ["prometheus"]
			processors: ["batch"]
			exporters: ["signozclickhousemetrics"]
		}
		logs: {
			receivers: ["otlp"]
			processors: ["batch"]
			exporters: ["clickhouselogsexporter"]
		}
	}
}

// Complete OTel configuration generator
#OtelCollector: {
	_config: #Config

	receivers:  #Receivers & {_config:  _config}
	processors: #Processors & {_config: _config}
	extensions: #Extensions & {_config: _config}
	exporters:  #Exporters & {_config:  _config}
	service:    #Service & {_config:    _config}
}