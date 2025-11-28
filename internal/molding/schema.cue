package schema

import (
    oc "github.com/signoz/foundry/moldings/otel_collector"
)

#SchemaVersion: =~"^v[0-9]+$"
#Platform: "docker" | "linux" | "kubernetes" | "aws" | "gcp" | "azure" | "windows"

#EnvVar: {
    key:   string
    value: string
}

// 1. OPEN Base: Defines the fields, but allows extension
_baseComponent: {
    enabled:  bool
    replicas: int & >=1
    version:  string & =~"^[0-9]+\\.[0-9]+(\\.[0-9]+)?(-.*)?$"
    env?: [...#EnvVar]
}

// 2. CLOSED Default: Used for standard components (signoz, etc.)
#Component: _baseComponent

// 3. CLOSED Specific: Used for otelCollector
#OtelCollectorComponent: _baseComponent & {
    config?: oc.#OtelCollector
}

// 4. Registry: ONLY contains special cases. NO [string] fallback here.
#ComponentRegistry: {
    otelCollector: #OtelCollectorComponent
}

_requirements: {
    docker:     ["docker", "docker-compose"]
    linux:      ["systemd", "curl", "tar"]
    kubernetes: ["kubectl", "helm"]
    aws:        ["aws-cli", "kubectl", "helm"]
    gcp:        ["gcloud", "kubectl", "helm"]
    azure:      ["az-cli", "kubectl", "helm"]
    windows:    ["powershell", "chocolatey"]
}

#Config: {
    schemaVersion: #SchemaVersion
    platform:      #Platform

    components: {
        // THE FIX:
        // Try to look up the ID in the Registry.
        // If found, use that schema.
        // If NOT found (lookup returns bottom _|_), fall back to #Component.
        [ID=string]: #ComponentRegistry[ID] | #Component
    }

    requirements: _requirements[platform]
}

#Config