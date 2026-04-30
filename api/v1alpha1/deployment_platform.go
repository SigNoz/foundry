package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*Platform)(nil)
var _ yaml.Unmarshaler = (*Platform)(nil)
var _ json.Marshaler = (*Platform)(nil)
var _ json.Unmarshaler = (*Platform)(nil)
var _ fmt.Stringer = (*Platform)(nil)
var _ jsonschema.Enum = (*Platform)(nil)

var (
	PlatformRender  Platform = Platform{s: "render"}
	PlatformCoolify Platform = Platform{s: "coolify"}
	PlatformRailway Platform = Platform{s: "railway"}
	PlatformECS     Platform = Platform{s: "ecs"}
	PlatformAWS     Platform = Platform{s: "aws"}
	PlatformGCP     Platform = Platform{s: "gcp"}
	PlatformAzure   Platform = Platform{s: "azure"}
)

type Platform struct {
	s string
}

func (platform Platform) String() string {
	return platform.s
}

func Platforms() []Platform {
	return []Platform{
		PlatformRender,
		PlatformCoolify,
		PlatformRailway,
		PlatformECS,
		PlatformAWS,
		PlatformGCP,
		PlatformAzure,
	}
}

func (platform Platform) MarshalJSON() ([]byte, error) {
	return json.Marshal(platform.String())
}

func (platform *Platform) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return platform.UnmarshalText([]byte(str))
}

func (platform *Platform) UnmarshalText(text []byte) error {
	for _, available := range Platforms() {
		if available.String() == string(text) {
			*platform = available
			return nil
		}
	}

	if len(text) == 0 {
		*platform = Platform{s: ""}
		return nil
	}

	return errors.New("invalid deployment platform: " + string(text))
}

func (platform Platform) MarshalText() ([]byte, error) {
	return []byte(platform.String()), nil
}

func (platform *Platform) UnmarshalYAML(node *yaml.Node) error {
	return platform.UnmarshalText([]byte(node.Value))
}

func (platform Platform) MarshalYAML() (any, error) {
	return platform.String(), nil
}

func (platform Platform) Enum() []any {
	platforms := []any{}
	for _, platform := range Platforms() {
		platforms = append(platforms, platform.String())
	}

	return platforms
}
