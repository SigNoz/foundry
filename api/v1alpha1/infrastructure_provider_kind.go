package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*InfrastructureProvider)(nil)
var _ yaml.Unmarshaler = (*InfrastructureProvider)(nil)
var _ json.Marshaler = (*InfrastructureProvider)(nil)
var _ json.Unmarshaler = (*InfrastructureProvider)(nil)
var _ fmt.Stringer = (*InfrastructureProvider)(nil)

var (
	InfrastructureProviderAWS   InfrastructureProvider = InfrastructureProvider{s: "aws"}
	InfrastructureProviderGCP   InfrastructureProvider = InfrastructureProvider{s: "gcp"}
	InfrastructureProviderAzure InfrastructureProvider = InfrastructureProvider{s: "azure"}
)

type InfrastructureProvider struct {
	s string
}

func (p InfrastructureProvider) String() string {
	return p.s
}

func (p InfrastructureProvider) IsZero() bool {
	return p.s == ""
}

func InfrastructureProviders() []InfrastructureProvider {
	return []InfrastructureProvider{InfrastructureProviderAWS, InfrastructureProviderGCP, InfrastructureProviderAzure}
}

func (p InfrastructureProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *InfrastructureProvider) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return p.UnmarshalText([]byte(str))
}

func (p *InfrastructureProvider) UnmarshalText(text []byte) error {
	for _, available := range InfrastructureProviders() {
		if available.String() == string(text) {
			*p = available
			return nil
		}
	}
	if len(text) == 0 {
		*p = InfrastructureProvider{s: ""}
		return nil
	}
	return errors.New("invalid infrastructure provider: " + string(text))
}

func (p InfrastructureProvider) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *InfrastructureProvider) UnmarshalYAML(node *yaml.Node) error {
	return p.UnmarshalText([]byte(node.Value))
}

func (p InfrastructureProvider) MarshalYAML() (any, error) {
	return p.String(), nil
}
