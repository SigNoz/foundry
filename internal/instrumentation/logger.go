package instrumentation

import (
	"log/slog"
	"os"
)

// NewLogger creates a new slog.Logger with pretty formatting.
// In debug mode, full timestamps and source locations are shown.
// In non-debug mode, compact output with just level and message is shown.
// Colors are automatically enabled when writing to a terminal.
func NewLogger(debug bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	return slog.New(newPrettyHandler(os.Stdout, &Options{
		AddSource:    true,
		Level:        level,
		Debug:        debug,
		ColorEnabled: isTTY(os.Stdout),
	}))
}

// isTTY reports whether the given file is a terminal.
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
