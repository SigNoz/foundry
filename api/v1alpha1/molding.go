package v1alpha1

import "maps"

type MoldingSpec struct {
	// Whether the molding is enabled
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty" description:"Whether the molding is enabled" default:"true"`

	// Cluster configuration for the molding
	Cluster TypeCluster `json:"cluster" yaml:"cluster,omitempty" description:"Cluster configuration for the molding"`

	// The version of the molding to use
	Version string `json:"version,omitempty" yaml:"version,omitempty" description:"The version of the molding to use" example:"latest"`

	// Image of the molding
	Image string `json:"image,omitempty" yaml:"image,omitempty" description:"Container image of the molding" example:"signoz/signoz:latest"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" description:"Environment variables for the molding"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty" description:"Configuration for the molding"`
}

type MoldingStatus struct {
	// Extra information about the molding
	Extras map[string]string `json:"extras,omitempty" yaml:"extras,omitempty" description:"Extra information about the molding"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" description:"Environment variables for the molding"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty" description:"Configuration for the molding"`
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
