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

// WithError records that the tracked operation failed and stores the typed
// error kind, its message, and the underlying cause so analytics can group
// failures by class while preserving the wrapped detail.
func (p Properties) WithError(err error) Properties {
	t, info, cause := errors.Unwrapb(err)
	p.values[propertyKeySuccess] = false
	p.values[propertyKeyErrorType] = t.String()
	p.values[propertyKeyError] = info
	if cause != nil {
		p.values[propertyKeyErrorCause] = cause.Error()
	}
	return p
}

// Map returns a copy of the underlying values, safe for callers to mutate.
func (p Properties) Map() map[string]any {
	out := make(map[string]any, len(p.values))
	maps.Copy(out, p.values)
	return out
}
