package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*TelemetryKeeperKind)(nil)
var _ yaml.Unmarshaler = (*TelemetryKeeperKind)(nil)
var _ json.Marshaler = (*TelemetryKeeperKind)(nil)
var _ json.Unmarshaler = (*TelemetryKeeperKind)(nil)
var _ fmt.Stringer = (*TelemetryKeeperKind)(nil)

var (
	TelemetryKeeperKindClickhouseKeeper TelemetryKeeperKind = TelemetryKeeperKind{s: "clickhousekeeper"}
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

func (kind TelemetryKeeperKind) MarshalYAML() (any, error) {
	return kind.String(), nil
}
