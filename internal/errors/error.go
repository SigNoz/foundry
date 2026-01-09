package errors

import "log/slog"

func LogAttr(err error) slog.Attr {
	return slog.Attr{
		Key:   "err",
		Value: slog.StringValue(err.Error()),
	}
}
