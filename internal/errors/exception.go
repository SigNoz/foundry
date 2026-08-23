package errors

import (
	"encoding/json"
	"log/slog"
)

type Exception struct {
	Type       string     `json:"type,omitempty"`
	Message    string     `json:"message"`
	Output     string     `json:"output,omitempty"`
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

// Output rides the outermost link that carries one: the tool's last words
// account for the whole conversation, and a further wrap must not drop them.
func ExceptionOf(err error) *Exception {
	e := exceptionOf(err)
	if e == nil {
		return nil
	}

	for link := err; link != nil; {
		b, ok := link.(*base)
		if !ok {
			break
		}

		if b.output != "" {
			e.Output = b.output
			break
		}

		link = b.cause
	}

	return e
}

// The walk terminates at the first non-*base link, which emits Message alone
// — stdlib wrappers format their full subtree in Error(), so re-walking them
// would duplicate that text. Stacktrace emits on TypeFatal links only; every
// *base captures one at construction time but emitting them all is noise.
func exceptionOf(err error) *Exception {
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
		Cause:   exceptionOf(b.cause),
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

	if e.Output != "" {
		attrs = append(attrs, slog.String("output", e.Output))
	}

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
