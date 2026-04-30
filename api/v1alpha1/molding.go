package v1alpha1

import "maps"

type MoldingSpec struct {
	// Whether the molding is enabled
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" description:"Whether the molding is enabled" default:"true"`

	// Cluster configuration for the molding
	Cluster TypeCluster `json:"cluster" yaml:"cluster,omitempty" description:"Cluster configuration for the molding"`

	// The version of the molding to use
	Version string `json:"version,omitempty" yaml:"version,omitempty" description:"The version of the molding to use" example:"latest"`

	// Image of the molding
	Image string `json:"image,omitempty" yaml:"image,omitempty" pattern:"^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*(:[a-zA-Z0-9._-]+)?(@sha256:[a-f0-9]{64})?$" description:"Container image of the molding" example:"signoz/signoz:latest"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" description:"Environment variables for the molding"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty" description:"Configuration for the molding"`

	_ struct{} `additionalProperties:"false"`
}

type MoldingStatus struct {
	// Extra information about the molding
	Extras map[string]string `json:"extras,omitempty" yaml:"extras,omitempty" description:"Extra information about the molding"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty" description:"Environment variables for the molding"`

	// Configuration for the molding
	Config TypeConfig `json:"config" yaml:"config,omitempty" description:"Configuration for the molding"`

	_ struct{} `additionalProperties:"false"`
}

// IsEnabled returns whether the molding is enabled.
// Returns false if the receiver or Enabled is nil.
func (spec *MoldingSpec) IsEnabled() bool {
	if spec == nil || spec.Enabled == nil {
		return false
	}
	return *spec.Enabled
}

func (spec *MoldingSpec) MergeStatus(status MoldingStatus) error {
	if spec.Env == nil {
		spec.Env = make(map[string]string)
	}

	if status.Env == nil {
		status.Env = make(map[string]string)
	}

	maps.Copy(status.Env, spec.Env)
	spec.Env = status.Env

	if err := Merge(&status.Config, spec.Config); err != nil {
		return err
	}

	spec.Config = status.Config

	return nil
}
