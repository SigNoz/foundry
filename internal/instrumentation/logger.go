package instrumentation

import (
	"log/slog"
	"os"
)

func NewLogger(debug bool) *slog.Logger {
	if debug {
		return newLoggerWithLevel(slog.LevelDebug)
	}

	return newLoggerWithLevel(slog.LevelInfo)
}

func newLoggerWithLevel(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: false,
	}))
}
