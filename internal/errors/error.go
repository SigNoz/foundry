package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log/slog"
)

type base struct {
	// t denotes the custom type of the error.
	t typ

	// info contains the error message
	info string

	// cause is the actual error which is being wrapped with a stacktrace and message information.
	cause error

	// s contains the stacktrace captured at error creation time.
	stacktrace fmt.Stringer
}

func (b *base) Error() string {
	if b.cause != nil {
		return fmt.Sprintf("%s: %s", b.info, b.cause.Error())
	}

	return b.info
}

func (b *base) WithStacktrace(stacktrace string) *base {
	b.stacktrace = rawStacktrace(stacktrace)
	return b
}

func (b *base) Stacktrace() string {
	return b.stacktrace.String()
}

func Newf(t typ, info string, args ...any) *base {
	return &base{
		t:          t,
		info:       fmt.Sprintf(info, args...),
		cause:      nil,
		stacktrace: newStackTrace(),
	}
}

func Wrapf(cause error, t typ, format string, args ...any) error {
	return &base{
		t:          t,
		info:       fmt.Sprintf(format, args...),
		cause:      cause,
		stacktrace: newStackTrace(),
	}
}

func Unwrapb(cause error) (typ, string, error) {
	base, ok := cause.(*base)
	if ok {
		return base.t, base.info, base.cause
	}

	return TypeInternal, cause.Error(), cause
}

// ExitCode returns the process exit code for err. nil is 0; an untyped error
// (anything not wrapping a *base via Newf/Wrapf) is 1; a typed error returns
// the code from its typ. The wrap chain is walked via errors.As so wrappers
// like fmt.Errorf("...%w", ...) don't lose the signal.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var b *base
	if stderrors.As(err, &b) {
		return b.t.ExitCode()
	}
	return 1
}

// TypeOf reports whether err (or anything in its wrap chain) was constructed
// via Newf or Wrapf, and returns its type. Distinguishes a typed TypeInternal
// from an untyped error — Unwrapb collapses both to TypeInternal, which loses
// the signal needed to map exit codes.
func TypeOf(err error) (typ, bool) {
	var b *base
	if stderrors.As(err, &b) {
		return b.t, true
	}
	return typ{}, false
}

func LogAttr(err error) slog.Attr {
	t, info, cause := Unwrapb(err)

	attrs := []slog.Attr{
		slog.String("type", t.String()),
		slog.String("message", info),
		slog.String("cause", cause.Error()),
	}

	type stacktracer interface {
		Stacktrace() string
	}

	if t == TypeFatal {
		if st, ok := err.(stacktracer); ok && st.Stacktrace() != "" {
			attrs = append(attrs, slog.String("stacktrace", st.Stacktrace()))
		}
	}

	if action := t.Action(); action != "" {
		attrs = append(attrs, slog.String("action", action))
	}

	return slog.GroupAttrs("exception", attrs...)
}

// JSONError is the JSON projection of a foundry error, suitable for emission
// on stdout under --format=json. Untyped errors collapse to ExitCode=1 with
// the original message and empty Type.
type JSONError struct {
	Type       string `json:"type"`
	ExitCode   int    `json:"exit_code"`
	Message    string `json:"message"`
	Cause      string `json:"cause,omitempty"`
	Action     string `json:"action,omitempty"`
	Stacktrace string `json:"stacktrace,omitempty"`
}

// Marshal implements writer.Outputable. The envelope key matches the slog
// group in LogAttr so log records and the --format=json stdout payload share
// the same field path.
func (e JSONError) Marshal() ([]byte, error) {
	return json.MarshalIndent(map[string]JSONError{"exception": e}, "", "  ")
}

// JSONErrorOf returns the JSON projection of err. nil returns a zero value.
// The wrap chain is walked via errors.As so wrappers like fmt.Errorf don't
// lose the signal. Stacktrace is populated only for TypeFatal, matching
// LogAttr.
func JSONErrorOf(err error) JSONError {
	if err == nil {
		return JSONError{}
	}
	var b *base
	if !stderrors.As(err, &b) {
		return JSONError{ExitCode: 1, Message: err.Error()}
	}
	je := JSONError{
		Type:     b.t.String(),
		ExitCode: b.t.ExitCode(),
		Message:  b.info,
		Action:   b.t.action,
	}
	if b.cause != nil {
		je.Cause = b.cause.Error()
	}
	if b.t == TypeFatal && b.stacktrace != nil {
		if st := b.stacktrace.String(); st != "" {
			je.Stacktrace = st
		}
	}
	return je
}
