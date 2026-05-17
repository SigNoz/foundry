package instrumentation

import (
	"log/slog"
	"os"
)

// NewLogger returns the process logger. When debug is false logs are dropped
// so the default UX matches helm/aws (silent on stdout/stderr, errors surface
// through the cobra exit path). When debug is true, debug-level records are
// emitted as JSON on stderr so stdout remains reserved for command results.
func NewLogger(debug bool) *slog.Logger {
	if !debug {
		return slog.New(slog.DiscardHandler)
	}

	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}))
}
