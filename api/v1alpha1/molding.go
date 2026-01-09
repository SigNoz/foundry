package v1alpha1

type MoldingSpec struct {
	// Cluster configuration for the molding
	Cluster TypeCluster `json:"cluster,omitempty" yaml:"cluster,omitempty"`

	// The version of the molding to use
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config,omitempty" yaml:"config,omitempty"`
}
