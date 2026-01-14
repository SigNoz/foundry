package v1alpha1

import (
	"encoding/json"
	"errors"

	"github.com/signoz/foundry/internal/types"
	"go.yaml.in/yaml/v3"
)

var _ json.Marshaler = (*TelemetryKeeperKind)(nil)
var _ json.Unmarshaler = (*TelemetryKeeperKind)(nil)

var (
	TelemetryKeeperKindClickhouseKeeper TelemetryKeeperKind = TelemetryKeeperKind{s: "clickhousekeeper"}
)

var (
	// TelemetryKeeperRaftAddresses is the key for inter-keeper consensus coordination.
	TelemetryKeeperRaftAddresses string = "raft"
	// TelemetryKeeperClientAddresses is the key for client connections.
	TelemetryKeeperClientAddresses string = "client"
)

type TelemetryKeeperKind struct {
	s string
}

func (kind TelemetryKeeperKind) String() string {
	return kind.s
}

func TelemetryKeeperKinds() []TelemetryKeeperKind {
	return []TelemetryKeeperKind{TelemetryKeeperKindClickhouseKeeper}
}

func (kind TelemetryKeeperKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *TelemetryKeeperKind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *TelemetryKeeperKind) UnmarshalText(text []byte) error {
	for _, availableKind := range TelemetryKeeperKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	if text == nil {
		*kind = TelemetryKeeperKind{s: ""}
		return nil
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
	Kind TelemetryKeeperKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the telemetry keeper.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func DefaultTelemetryKeeper() TelemetryKeeper {
	return TelemetryKeeper{
		Kind: TelemetryKeeperKindClickhouseKeeper,
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-keeper:25.5.6",
		},
	}
}
