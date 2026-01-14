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
	// Whether the molding is enabled
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Cluster configuration for the molding
	Cluster TypeCluster `json:"cluster,omitempty" yaml:"cluster,omitempty"`

	// The version of the molding to use
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Image of the molding
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

type MoldingStatus struct {
	// Status of the molding
	Addresses map[string][]string `json:"addresses,omitempty" yaml:"addresses,omitempty"`

	// Extra information about the molding
	Extras map[string]string `json:"extras,omitempty" yaml:"extras,omitempty"`

	// Environment variables for the molding
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// Configuration for the molding
	Config TypeConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

func (spec *MoldingSpec) MergeStatus(status MoldingStatus) error {
	if spec.Env == nil {
		spec.Env = make(map[string]string)
	}

	if status.Env == nil {
		status.Env = make(map[string]string)
	}

	for key, value := range status.Env {
		spec.Env[key] = value
	}

	if err := Merge(&spec.Config, status.Config); err != nil {
		return err
	}

	return nil
}
