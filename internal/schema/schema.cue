package schema

import (
	oc "github.com/signoz/foundry/moldings/otel_collector"
)

// Schema version must follow semantic versioning
#SchemaVersion: =~"^v[0-9]+$"

// Platform enumeration - add more platforms as needed
#Platform: "docker" | "linux" | "kubernetes" | "aws" | "gcp" | "azure" | "windows"

// Environment variable key-value pair
#EnvVar: {
	key:   string
	value: string
}

// Component definition
#Component: {
	enabled:  bool
	replicas: int & >=1
	version:  string & =~"^[0-9]+\\.[0-9]+(\\.[0-9]+)?(-.*)?$"
	env?: [...#EnvVar]
}

// Otel Collector Defination
#OtelCollectorComponent: {
	#Component

	config: oc.#OtelCollector
}
// Platform-specific requirements
_requirements: {
	docker: ["docker", "docker-compose"]
	linux: ["systemd", "curl", "tar"]
	kubernetes: ["kubectl", "helm"]
	aws: ["aws-cli", "kubectl", "helm"]
	gcp: ["gcloud", "kubectl", "helm"]
	azure: ["az-cli", "kubectl", "helm"]
	windows: ["powershell", "chocolatey"]
}

// Main configuration schema
#Config: {
	schemaVersion: #SchemaVersion
	platform:      #Platform

	// Components involved in the deployment
	components: {
		[string]: #Component
		"otel-collector"?: #OtelCollectorComponent
	}

	// Requirements based on platform
	requirements: _requirements[platform]
}

// Validate that the config conforms to the schema
#Config
