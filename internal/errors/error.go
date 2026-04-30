package errors

import (
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

		attrs = append(attrs, slog.String("action", "Please raise an issue at https://github.com/signoz/foundry/issues with the error message and stacktrace."))
	}

	if t == TypeUnsupported {
		attrs = append(attrs, slog.String("action", "Please check the documentation for supported features or raise an issue at https://github.com/signoz/foundry/issues for feature requests."))
	}

	return slog.GroupAttrs("exception", attrs...)
}
