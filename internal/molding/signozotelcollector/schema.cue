package signozotelcollector

// Common Types
#Endpoint: string & =~"^[^:]+:[0-9]+$"

#TLSConfig: {
	cert_file?: string
	key_file?:  string
	ca_file?:   string
	insecure?:  bool
}

// OTLP Receiver
#GRPCProtocol: {
	endpoint?:               #Endpoint
	transport?:              string & =~"^(tcp|unix)$"
	max_recv_msg_size_mib?:  int & >0
	max_concurrent_streams?: uint
	tls?:                    #TLSConfig
}

#HTTPProtocol: {
	endpoint?:              #Endpoint
	max_request_body_size?: int & >0
	tls?:                   #TLSConfig
}

#OTLPReceiver: {
	protocols?: {
		grpc?: #GRPCProtocol
		http?: #HTTPProtocol
	}
}

// Prometheus Receiver
#ScrapeConfig: {
	job_name:         string
	scrape_interval?: string & =~"^[0-9]+(s|m|h)$"
	static_configs?: [...{
		targets: [...string]
		labels?: [string]: string
	}]
}

#PrometheusReceiver: {
	config?: {
		global?: {
			scrape_interval?: string & =~"^[0-9]+(s|m|h)$"
		}
		scrape_configs?: [...#ScrapeConfig]
	}
}

// Top-level Sections
#Receivers: {
	otlp?:       #OTLPReceiver
	prometheus?: #PrometheusReceiver
	[string]: {...}
}

#Processors: {[string]: {...}}
#Exporters: {[string]: {...}}
#Extensions: {[string]: {...}}
#Connectors: {[string]: {...}}

#Pipeline: {
	receivers?: [...string]
	processors?: [...string]
	exporters?: [...string]
}

#Pipelines: {[string]: #Pipeline}

#Service: {
	telemetry?: {
		logs?: {
			encoding?: string
		}
	}
	extensions?: [...string]
	pipelines?: #Pipelines
}

#ConfigSpec: {
	connectors?: #Connectors
	receivers?:  #Receivers
	processors?: #Processors
	exporters?:  #Exporters
	extensions?: #Extensions
	service?:    #Service
}
