package v1alpha1

import (
	"encoding/json"
	"errors"

	"github.com/signoz/foundry/internal/types"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*TelemetryStoreKind)(nil)
var _ yaml.Unmarshaler = (*TelemetryStoreKind)(nil)

var _ json.Marshaler = (*TelemetryStoreKind)(nil)
var _ json.Unmarshaler = (*TelemetryStoreKind)(nil)

var (
	TelemetryStoreKindClickhouse TelemetryStoreKind = TelemetryStoreKind{s: "clickhouse"}
)

var (
	// TelemetryStoreClusterAddresses is the key for cluster node addresses.
	TelemetryStoreClusterAddresses string = "cluster"
)

type TelemetryStoreKind struct {
	s string
}

func TelemetryStoreKinds() []TelemetryStoreKind {
	return []TelemetryStoreKind{TelemetryStoreKindClickhouse}
}

func (kind TelemetryStoreKind) String() string {
	return kind.s
}

func (kind TelemetryStoreKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(kind.String())
}

func (kind *TelemetryStoreKind) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return kind.UnmarshalText([]byte(str))
}

func (kind *TelemetryStoreKind) UnmarshalText(text []byte) error {
	for _, availableKind := range TelemetryStoreKinds() {
		if availableKind.String() == string(text) {
			*kind = availableKind
			return nil
		}
	}
	if text == nil {
		*kind = TelemetryStoreKind{s: ""}
		return nil
	}
	return errors.New("invalid telemetry store kind: " + string(text))
}

func (kind TelemetryStoreKind) MarshalText() ([]byte, error) {
	return []byte(kind.String()), nil
}

func (kind *TelemetryStoreKind) UnmarshalYAML(node *yaml.Node) error {
	return kind.UnmarshalText([]byte(node.Value))
}

func (kind TelemetryStoreKind) MarshalYAML() (interface{}, error) {
	return kind.String(), nil
}

type TelemetryStore struct {
	// Kind of the telemetry store to use.
	Kind TelemetryStoreKind `json:"kind,omitzero" yaml:"kind,omitempty"`

	// Specification for the telemetry store.
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func DefaultTelemetryStore() TelemetryStore {
	return TelemetryStore{
		Kind: TelemetryStoreKindClickhouse,
		Spec: MoldingSpec{
			Enabled: true,
			Cluster: TypeCluster{
				Replicas: types.NewIntPtr(0),
				Shards:   types.NewIntPtr(1),
			},
			Version: "25.5.6",
			Image:   "clickhouse/clickhouse-server:25.5.6",
		},
	}
}
