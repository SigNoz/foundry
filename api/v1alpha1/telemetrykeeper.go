package v1alpha1

import (
	"errors"

	"go.yaml.in/yaml/v3"
)

var (
	TelemetryKeeperKindClickhouse TelemetryKeeperKind = TelemetryKeeperKind{s: "clickhousekeeper"}
)

type TelemetryKeeperKind struct {
	s string
}

func (kind TelemetryKeeperKind) String() string {
	return kind.s
}

func TelemetryKeeperKinds() []TelemetryKeeperKind {
	return []TelemetryKeeperKind{TelemetryKeeperKindClickhouse}
}

func (kind *TelemetryKeeperKind) UnmarshalText(text []byte) error {
	for _, availableKind := range TelemetryKeeperKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	return errors.New("invalid telemetry keeper kind: " + string(text))
}

func (kind TelemetryKeeperKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *TelemetryKeeperKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind TelemetryKeeperKind) MarshalYAML() (interface{}, error) {
	return kind.String(), nil
}

type TelemetryKeeper struct {
	// Kind of the telemetry keeper to use.
	Kind TelemetryKeeperKind `json:"kind,omitempty" yaml:"kind,omitempty"`

	// Specification for the telemetry keeper.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (t TelemetryKeeper) MergeStatusIntoSpec() TelemetryKeeper {
	return TelemetryKeeper{
		Kind:   t.Kind,
		Spec:   MergeStatusIntoSpec(t.Spec, t.Status),
		Status: t.Status,
	}
}
