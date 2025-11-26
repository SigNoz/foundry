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
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}))
}
