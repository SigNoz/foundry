package v1alpha1

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*MoldingKind)(nil)
var _ yaml.Unmarshaler = (*MoldingKind)(nil)
var _ fmt.Stringer = (*MoldingKind)(nil)

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

func (kind MoldingKind) MarshalYAML() (any, error) {
	return kind.String(), nil
}
