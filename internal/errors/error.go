package errors

import (
	goerrors "errors"
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

func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var b *base
	if goerrors.As(err, &b) {
		return b.t.ExitCode()
	}

	return 1
}

func LogAttr(err error) slog.Attr {
	return slog.GroupAttrs("exception", exceptionAttrs(ExceptionOf(err))...)
}
