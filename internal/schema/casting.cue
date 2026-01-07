package casting

import (
	signozOtelCollectorConfig "github.com/signoz/foundry/molding/signozotelcollector"
	clickhouseConfig "github.com/signoz/foundry/molding/clickhouse"
	signozConfig "github.com/signoz/foundry/molding/signoz"
	zookeeperConfig "github.com/signoz/foundry/molding/zookeeper"
	postgresConfig "github.com/signoz/foundry/molding/postgres"

)

// Matches versions like v1, v2, v3
#SchemaVersion: =~"^v[0-9]+$"

// Supported platform identifiers
#Platform: "docker" | "linux" | "kubernetes" | "aws" | "gcp" | "azure" | "windows"

// Base definition for a deployable component.
// Common fields shared across all components.
_baseComponent: {
	enabled:  bool
	replicas: int & >=1
	version:  string & =~"^[0-9]+\\.[0-9]+(\\.[0-9]+)?(-.*)?$"
}

// Generic component definition used when a component
// does not require any special fields.
#Component: _baseComponent

// Known components that have special schemas.
// Components listed here override the generic component definition.
#ComponentRegistry: {
	signozOtelCollector: _baseComponent & {
		config: signozOtelCollectorConfig.#BaseConfig
	}
	clickhouse: _baseComponent & {
		config: clickhouseConfig.#BaseConfig
	}
	signoz: _baseComponent & {
		config: signozConfig.#BaseConfig
	}
	zookeeper: _baseComponent & {
		config: zookeeperConfig.#BaseConfig
	}
	postgres: _baseComponent & {
		config: postgresConfig.#BaseConfig
	}
}

// Platform-specific external requirements.
_requirements: {
	docker: ["docker", "docker-compose"]
	linux: ["systemd", "curl", "tar"]
	kubernetes: ["kubectl", "helm"]
	aws: ["aws-cli", "kubectl", "helm"]
	gcp: ["gcloud", "kubectl", "helm"]
	azure: ["az-cli", "kubectl", "helm"]
	windows: ["powershell", "chocolatey"]
}

// Top-level configuration schema for a Foundry casting.
// Selects the appropriate component schema based on ID.
// Falls back to the generic component definition if not matched.
#Config: {
	schemaVersion: #SchemaVersion
	platform:      #Platform

	components: {
		[ID=string]: #ComponentRegistry[ID]
	}

	requirements: _requirements[platform]
}

#Config
