package domain

import (
	"maps"

	"github.com/signoz/foundry/internal/errors"
)

const (
	propertyKeySuccess    = "success"
	propertyKeyError      = "error"
	propertyKeyErrorType  = "error_type"
	propertyKeyErrorCause = "error_cause"
)

// Properties is a string-keyed bag of telemetry values with a fixed shape for
// the success/error envelope (see WithSuccess and WithError). Set, WithSuccess,
// and WithError mutate the underlying map and return the receiver for chaining.
type Properties struct {
	values map[string]any
}

func NewProperties() Properties {
	return Properties{values: make(map[string]any)}
}

func (p Properties) Set(key string, value any) Properties {
	p.values[key] = value
	return p
}

// WithSuccess records that the tracked operation succeeded.
func (p Properties) WithSuccess() Properties {
	p.values[propertyKeySuccess] = true
	return p
}

// WithError records the typed kind, the outer message, and the next link's
// own Message as the cause — using the link's Message (not the chained
// Error()) gives stable grouping keys for analytics.
func (p Properties) WithError(err error) Properties {
	p.values[propertyKeySuccess] = false

	e := errors.ExceptionOf(err)
	if e == nil {
		return p
	}

	p.values[propertyKeyErrorType] = e.Type
	p.values[propertyKeyError] = e.Message
	if e.Cause != nil {
		p.values[propertyKeyErrorCause] = e.Cause.Message
	}

	return p
}

// Map returns a copy of the underlying values, safe for callers to mutate.
func (p Properties) Map() map[string]any {
	out := make(map[string]any, len(p.values))
	maps.Copy(out, p.values)
	return out
}
