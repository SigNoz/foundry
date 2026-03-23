package v1alpha1

import "encoding/json"

// Infrastructure holds the configuration for infrastructure manifest generation (e.g., Terraform).
// The compute type is resolved automatically by foundry based on the provider and deployment mode —
// users only need to specify the cloud provider.
type Infrastructure struct {
	// Whether infrastructure manifest generation is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// The cloud provider to generate infrastructure manifests for (aws, gcp, azure)
	Provider InfrastructureProvider `json:"provider" yaml:"provider"`

	// Status holds the generated IaC file contents keyed by filename (e.g. "main.tf").
	// This is populated by foundry after generation and written to the lock file.
	Status map[string]string `json:"status,omitempty" yaml:"status,omitempty"`
}

// MarshalJSON implements json.Marshaler. It manually omits Provider and Status when zero
// so that the strategic merge patch doesn't overwrite defaults with empty values.
// Go's encoding/json omitempty does not omit zero-value struct fields, only basic types.
func (i Infrastructure) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"enabled": i.Enabled,
	}
	if !i.Provider.IsZero() {
		m["provider"] = i.Provider.String()
	}
	if len(i.Status) > 0 {
		m["status"] = i.Status
	}
	return json.Marshal(m)
}

// DefaultInfrastructure returns the default Infrastructure configuration.
func DefaultInfrastructure() Infrastructure {
	return Infrastructure{
		Enabled:  false,
		Provider: InfrastructureProviderAWS,
	}
}
