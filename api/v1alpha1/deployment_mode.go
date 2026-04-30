package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*Mode)(nil)
var _ yaml.Unmarshaler = (*Mode)(nil)
var _ json.Marshaler = (*Mode)(nil)
var _ json.Unmarshaler = (*Mode)(nil)
var _ fmt.Stringer = (*Mode)(nil)
var _ jsonschema.Enum = (*Mode)(nil)

var (
	ModeDocker     Mode = Mode{s: "docker"}
	ModeSystemd    Mode = Mode{s: "systemd"}
	ModeKubernetes Mode = Mode{s: "kubernetes"}
	ModeEC2        Mode = Mode{s: "ec2"}
)

type Mode struct {
	s string
}

func (mode Mode) String() string {
	return mode.s
}

func Modes() []Mode {
	return []Mode{ModeDocker, ModeSystemd, ModeKubernetes, ModeEC2}
}

func (mode Mode) MarshalJSON() ([]byte, error) {
	return json.Marshal(mode.String())
}

func (mode *Mode) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return mode.UnmarshalText([]byte(str))
}

func (mode *Mode) UnmarshalText(text []byte) error {
	for _, available := range Modes() {
		if available.String() == string(text) {
			*mode = available
			return nil
		}
	}

	if len(text) == 0 {
		*mode = Mode{s: ""}
		return nil
	}

	return errors.New("invalid deployment mode: " + string(text))
}

func (mode Mode) MarshalText() ([]byte, error) {
	return []byte(mode.String()), nil
}

func (mode *Mode) UnmarshalYAML(node *yaml.Node) error {
	return mode.UnmarshalText([]byte(node.Value))
}

func (mode Mode) MarshalYAML() (any, error) {
	return mode.String(), nil
}

func (mode Mode) Enum() []any {
	modes := []any{}
	for _, mode := range Modes() {
		modes = append(modes, mode.String())
	}

	return modes
}
