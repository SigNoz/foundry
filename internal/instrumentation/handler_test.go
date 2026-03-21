package instrumentation

import (
	"bytes"
	"context"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrettyHandler(t *testing.T) {
	testCases := []struct {
		name     string
		f        func(logger *slog.Logger)
		opts     *Options
		expected string
	}{
		{
			name: "DebugModeSimple",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "this is a pretty log message")
			},
			opts: &Options{
				Level:     slog.LevelDebug,
				AddSource: true,
				Debug:     true,
			},
			expected: `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [-+][0-9]{2}:[0-9]{2} \| INFO   \| internal/instrumentation\.TestPrettyHandler\.func1:\d+ - this is a pretty log message\n`,
		},
		{
			name: "DebugModeWithAttrs",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "this is a pretty log message with attrs", slog.String("k", "v"))
			},
			opts: &Options{
				Level:     slog.LevelDebug,
				AddSource: true,
				Debug:     true,
			},
			expected: `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [-+][0-9]{2}:[0-9]{2} \| INFO   \| internal/instrumentation\.TestPrettyHandler\.func2:\d+ - this is a pretty log message with attrs k=v\n`,
		},
		{
			name: "CompactModeSimple",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "loading casting.yaml")
			},
			opts: &Options{
				Level: slog.LevelInfo,
				Debug: false,
			},
			expected: `^ INFO   loading casting\.yaml\n$`,
		},
		{
			name: "CompactModeWithAttrs",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "parsed casting", slog.String("name", "signoz"), slog.String("mode", "docker"))
			},
			opts: &Options{
				Level: slog.LevelInfo,
				Debug: false,
			},
			expected: `^ INFO   parsed casting name=signoz mode=docker\n$`,
		},
		{
			name: "CompactModeWarnLevel",
			f: func(logger *slog.Logger) {
				logger.WarnContext(context.Background(), "something is off")
			},
			opts: &Options{
				Level: slog.LevelInfo,
				Debug: false,
			},
			expected: `^ WARN   something is off\n$`,
		},
		{
			name: "CompactModeErrorLevel",
			f: func(logger *slog.Logger) {
				logger.ErrorContext(context.Background(), "failed to forge")
			},
			opts: &Options{
				Level: slog.LevelInfo,
				Debug: false,
			},
			expected: `^ ERROR  failed to forge\n$`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(newPrettyHandler(&buf, tc.opts))
			tc.f(logger)

			re, err := regexp.Compile(tc.expected)
			require.NoError(t, err)

			assert.Regexp(t, re, buf.String())
		})
	}
}

func TestPrettyHandlerNoColorWhenDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newPrettyHandler(&buf, &Options{
		Level:        slog.LevelInfo,
		Debug:        false,
		ColorEnabled: false,
	}))
	logger.InfoContext(context.Background(), "no color here")
	output := buf.String()

	// Verify no ANSI escape codes in output
	assert.False(t, strings.Contains(output, "\033["), "output should not contain ANSI escape codes when color is disabled")
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "no color here")
}

func TestPrettyHandlerColorWhenEnabled(t *testing.T) {
	// Force fatih/color to output colors even when not a TTY (for testing)
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = true })

	var buf bytes.Buffer
	logger := slog.New(newPrettyHandler(&buf, &Options{
		Level:        slog.LevelInfo,
		Debug:        false,
		ColorEnabled: true,
	}))
	logger.InfoContext(context.Background(), "colored output", slog.String("key", "val"))
	output := buf.String()

	// Verify ANSI escape codes are present for colored level
	assert.True(t, strings.Contains(output, "\033["), "output should contain ANSI escape codes when color is enabled")
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "colored output")
}

func TestPrettyHandlerCompactNoTimestampOrSource(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newPrettyHandler(&buf, &Options{
		Level:     slog.LevelInfo,
		AddSource: true,
		Debug:     false,
	}))
	logger.InfoContext(context.Background(), "compact test")
	output := buf.String()

	// In compact mode, no timestamp or source should appear
	assert.NotContains(t, output, "|", "compact mode should not contain pipe separators")
	assert.NotContains(t, output, "-", "compact mode should not contain dash separator from source")
}

func TestPrettyHandlerDebugShowsTimestampAndSource(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newPrettyHandler(&buf, &Options{
		Level:     slog.LevelDebug,
		AddSource: true,
		Debug:     true,
	}))
	logger.InfoContext(context.Background(), "debug test")
	output := buf.String()

	// In debug mode, timestamp and source should be present
	assert.Contains(t, output, "|", "debug mode should contain pipe separators")
	assert.Contains(t, output, "instrumentation.TestPrettyHandlerDebugShowsTimestampAndSource", "debug mode should contain source function name")
}
