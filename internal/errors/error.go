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

func (b *base) Unwrap() error {
	return b.cause
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

// JSONError is the recursive JSON projection of a foundry error. Cause is
// the next link in the wrap chain rendered with the same shape, so agents
// can recurse without a different schema at each depth. The walk terminates
// at the first non-foundry link, which emits Message alone — stdlib wrappers
// format their full subtree in Error(), so re-walking them would duplicate
// that text. The top-level emission envelope is added by JSONException, not
// this type, so nested causes don't accidentally re-wrap themselves.
type JSONError struct {
	Type       string     `json:"type,omitempty"`
	Message    string     `json:"message"`
	Cause      *JSONError `json:"cause,omitempty"`
	Action     string     `json:"action,omitempty"`
	Stacktrace string     `json:"stacktrace,omitempty"`
}

// JSONException is the top-level stream emission envelope for a JSONError.
// The "exception" key matches the slog group in LogAttr so log records and
// the --format=json stdout payload share the same field path.
type JSONException struct {
	Error JSONError
}

func (e JSONException) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(map[string]JSONError{"exception": e.Error}, "", "  ")
}

// JSONErrorOf returns the emission envelope for err. nil yields a zero
// envelope. The outermost link populates the top-level fields; the rest of
// the chain is rendered recursively under Cause. Stacktrace is populated
// only for TypeFatal, matching LogAttr.
func JSONErrorOf(err error) JSONException {
	if err == nil {
		return JSONException{}
	}

	je := *projectError(err)

	var b *base
	if stderrors.As(err, &b) && b.t == TypeFatal && b.stacktrace != nil {
		if st := b.stacktrace.String(); st != "" {
			je.Stacktrace = st
		}
	}

	return JSONException{Error: je}
}

// projectError renders one link of the wrap chain. *base links carry their
// own info/type/action (info, not Error(), so the subtree isn't re-emitted)
// and the walk continues at b.cause. The first non-*base link emits Message
// alone and terminates the walk — stdlib wrappers format their own chain in
// Error(), so re-walking via errors.Unwrap would duplicate that text.
func projectError(err error) *JSONError {
	if err == nil {
		return nil
	}

	b, ok := err.(*base)
	if !ok {
		return &JSONError{Message: err.Error()}
	}

	return &JSONError{
		Type:    b.t.String(),
		Message: b.info,
		Action:  b.t.action,
		Cause:   projectError(b.cause),
	}
}
