package v1alpha1

import "encoding/json"

// Infrastructure holds the configuration for infrastructure manifest generation (e.g., Terraform).
// The cloud provider is resolved automatically from spec.deployment.platform — no provider field
// is needed here.
type Infrastructure struct {
	// Whether infrastructure manifest generation is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Status holds the generated IaC file contents keyed by filename (e.g. "main.tf.json").
	// This is populated by foundry after generation and written to the lock file.
	Status map[string]string `json:"status,omitempty" yaml:"status,omitempty"`

	_ struct{} `additionalProperties:"false"`
}

// MarshalJSON implements json.Marshaler. It manually omits Status when zero
// so that the strategic merge patch doesn't overwrite defaults with empty values.
func (i Infrastructure) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"enabled": i.Enabled,
	}
	if len(i.Status) > 0 {
		m["status"] = i.Status
	}
	return json.Marshal(m)
}

// DefaultInfrastructure returns the default Infrastructure configuration.
func DefaultInfrastructure() Infrastructure {
	return Infrastructure{
		Enabled: false,
	}
}
