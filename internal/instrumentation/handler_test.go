package instrumentation

import (
	"bytes"
	"context"
	"log/slog"
	"regexp"
	"testing"

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
			name: "Simple",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "this is a pretty log message")
			},
			opts: &Options{
				Level:     slog.LevelDebug,
				AddSource: true,
			},
			expected: `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [-+][0-9]{2}:[0-9]{2} \| INFO   \| internal/instrumentation\.TestPrettyHandler\.func1:24 - this is a pretty log message\n`,
		},
		{
			name: "WithAttrs",
			f: func(logger *slog.Logger) {
				logger.InfoContext(context.Background(), "this is a pretty log message with attrs", slog.String("k", "v"))
			},
			opts: &Options{
				Level:     slog.LevelDebug,
				AddSource: true,
			},
			expected: `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} [-+][0-9]{2}:[0-9]{2} \| INFO   \| internal/instrumentation\.TestPrettyHandler\.func2:35 - this is a pretty log message with attrs k=v\n`,
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
