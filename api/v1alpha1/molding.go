package v1alpha1

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

var (
	MoldingKindIngester        MoldingKind = MoldingKind{s: "ingester"}
	MoldingKindTelemetryStore  MoldingKind = MoldingKind{s: "telemetrystore"}
	MoldingKindTelemetryKeeper MoldingKind = MoldingKind{s: "telemetrykeeper"}
	MoldingKindMetaStore       MoldingKind = MoldingKind{s: "metastore"}
	MoldingKindSignoz          MoldingKind = MoldingKind{s: "signoz"}
)

type MoldingKind struct {
	s string
}

func (kind MoldingKind) String() string {
	return kind.s
}

func MoldingKinds() []MoldingKind {
	return []MoldingKind{MoldingKindIngester, MoldingKindTelemetryStore, MoldingKindTelemetryKeeper, MoldingKindMetaStore, MoldingKindSignoz}
}

func (kind *MoldingKind) UnmarshalText(text []byte) error {
	for _, availableKind := range MoldingKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	return errors.New("invalid molding kind: " + string(text))
}

func (kind MoldingKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *MoldingKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind MoldingKind) MarshalYAML() (interface{}, error) {
	return kind.String(), nil
}

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

type MoldingStatus struct {
	// Status of the molding
	Addresses []string `json:"addresses,omitempty" yaml:"addresses,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

func MergeStatusIntoSpec(spec MoldingSpec, status MoldingStatus) MoldingSpec {
	mergedEnv := make(map[string]string)
	for k, v := range spec.Env {
		mergedEnv[k] = v
	}

	for k, v := range status.Env {
		mergedEnv[k] = v
	}

	mergedConfig := make(map[string]string)
	for k, v := range spec.Config.Data {
		mergedConfig[k] = v
	}
	for k, v := range status.Config.Data {
		mergedConfig[k] = v
	}

	return MoldingSpec{
		Cluster: spec.Cluster,
		Version: spec.Version,
		Env:     mergedEnv,
		Config:  TypeConfig{Data: mergedConfig},
	}
}
