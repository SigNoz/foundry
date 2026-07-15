package v1alpha1

import "encoding/json"

// TypeCastingRef references a casting: the identity it is declared with and
// the casting it resolved to.
type TypeCastingRef struct {
	Spec   TypeCastingRefSpec   `json:"spec" yaml:"spec" required:"true" description:"Identity of the referenced casting"`
	Status TypeCastingRefStatus `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the reference"`
	_      struct{}             `additionalProperties:"false"`
}

type TypeCastingRefSpec struct {
	TypeVersion `json:",inline" yaml:",inline"`

	Name string   `json:"name" yaml:"name" required:"true" nullable:"false" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"Name of the referenced casting."`
	_    struct{} `additionalProperties:"false"`
}

// MarshalJSON implements json.Marshaler. It manually omits empty fields
// so that the strategic merge patch doesn't overwrite defaults with empty
// values.
func (spec TypeCastingRefSpec) MarshalJSON() ([]byte, error) {
	m := map[string]any{}
	if spec.APIVersion != "" {
		m["apiVersion"] = spec.APIVersion
	}
	if spec.Name != "" {
		m["name"] = spec.Name
	}
	return json.Marshal(m)
}

type TypeCastingRefStatus struct {
	Status `json:",inline" yaml:",inline"`

	Casting string   `json:"casting,omitempty" yaml:"casting,omitempty" description:"The resolved casting, verbatim"`
	_       struct{} `additionalProperties:"false"`
}
