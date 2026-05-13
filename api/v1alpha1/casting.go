package v1alpha1

import (
	"encoding/json"
	"fmt"

	"github.com/signoz/foundry/internal/domain"
	"github.com/swaggest/jsonschema-go"
)

// Casting is the envelope. Kind discriminates the concrete shape held in
// Spec and Status (as pointers, so mutations propagate). Per-kind types,
// accessors, and defaults live in casting_<kind>.go.
type Casting struct {
	TypeVersion `json:",inline" yaml:",inline"`
	Kind        Kind         `json:"kind" yaml:"kind" description:"Kind of the casting resource. Defaults to SigNoz when omitted."`
	Metadata    TypeMetadata `json:"metadata" yaml:"metadata" required:"true" description:"Metadata of the casting configuration"`
	Spec        any          `json:"spec" yaml:"spec" required:"true" description:"Specification for the casting"`
	Status      any          `json:"status,omitzero" yaml:"status,omitempty" description:"Status of the casting"`
	_           struct{}     `additionalProperties:"false"`
}

var _ jsonschema.OneOfExposer = Casting{}

// JSONSchemaOneOf lists the schema-only variant types per kind. Adding a new
// kind = one entry here plus its casting_<kind>.go file.
func (Casting) JSONSchemaOneOf() []any {
	return []any{
		kindSigNozCasting{},
	}
}

// UnmarshalJSON decodes envelope fields via a method-stripped alias, then
// dispatches Spec and Status into the right concrete pointer-typed value
// based on Kind.
func (c *Casting) UnmarshalJSON(data []byte) error {
	type castingAlias Casting
	if err := json.Unmarshal(data, (*castingAlias)(c)); err != nil {
		return err
	}

	if c.Kind == (Kind{}) {
		c.Kind = KindSigNoz
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch c.Kind {
	case KindSigNoz:
		return c.unmarshalSigNozSpecAndStatus(raw)
	default:
		return fmt.Errorf("unknown casting kind %q", c.Kind)
	}
}

// TrackableProperties extracts analytics properties from the casting.
// Dispatches on the concrete Spec type; envelope-only properties are returned
// when the kind has no registered analytics handler.
func (c Casting) TrackableProperties() domain.Properties {
	props := domain.NewProperties().Set("kind", c.Kind.String())

	switch spec := c.Spec.(type) {
	case *SigNozCastingSpec:
		return signozTrackableProperties(props, spec)
	}
	return props
}
