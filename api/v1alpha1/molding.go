package v1alpha1

import "maps"

type MoldingSpec struct {
	// Whether the molding is enabled
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Cluster configuration for the molding
	Cluster TypeCluster `json:"cluster" yaml:"cluster,omitempty"`

	// The version of the molding to use
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Image of the molding
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty"`
}

type MoldingStatus struct {
	// Extra information about the molding
	Extras map[string]string `json:"extras,omitempty" yaml:"extras,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty"`
}

func (spec *MoldingSpec) MergeStatus(status MoldingStatus) error {
	if spec.Env == nil {
		spec.Env = make(map[string]string)
	}

	if status.Env == nil {
		status.Env = make(map[string]string)
	}

	maps.Copy(spec.Env, status.Env)

	if err := Merge(&spec.Config, status.Config); err != nil {
		return err
	}

	return nil
}
