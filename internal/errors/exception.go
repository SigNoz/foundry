package errors

import (
	"encoding/json"
	"log/slog"
)

type Exception struct {
	Type       string     `json:"type,omitempty"`
	Message    string     `json:"message"`
	Cause      *Exception `json:"cause,omitempty"`
	Action     string     `json:"action,omitempty"`
	Stacktrace string     `json:"stacktrace,omitempty"`
}

type Envelope struct {
	Exception *Exception
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(map[string]*Exception{"exception": e.Exception}, "", "  ")
}

// The walk terminates at the first non-*base link, which emits Message alone
// — stdlib wrappers format their full subtree in Error(), so re-walking them
// would duplicate that text. Stacktrace emits on TypeFatal links only; every
// *base captures one at construction time but emitting them all is noise.
func ExceptionOf(err error) *Exception {
	if err == nil {
		return nil
	}

	b, ok := err.(*base)
	if !ok {
		return &Exception{Message: err.Error()}
	}

	e := &Exception{
		Type:    b.t.String(),
		Message: b.info,
		Action:  b.t.action,
		Cause:   ExceptionOf(b.cause),
	}
	if b.t == TypeFatal && b.stacktrace != nil {
		if st := b.stacktrace.String(); st != "" {
			e.Stacktrace = st
		}
	}

	return e
}

func EnvelopeOf(err error) Envelope {
	return Envelope{Exception: ExceptionOf(err)}
}

func exceptionAttrs(e *Exception) []slog.Attr {
	if e == nil {
		return nil
	}

	var attrs []slog.Attr
	if e.Type != "" {
		attrs = append(attrs, slog.String("type", e.Type))
	}

	attrs = append(attrs, slog.String("message", e.Message))
	if e.Cause != nil {
		attrs = append(attrs, slog.GroupAttrs("cause", exceptionAttrs(e.Cause)...))
	}

	if e.Action != "" {
		attrs = append(attrs, slog.String("action", e.Action))
	}

	if e.Stacktrace != "" {
		attrs = append(attrs, slog.String("stacktrace", e.Stacktrace))
	}

	return attrs
}
