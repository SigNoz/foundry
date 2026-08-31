package tooler

import (
	"bytes"
	"context"
	"testing"

	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func binary(t *testing.T, name string) string {
	t.Helper()

	path, err := Resolve(name)
	require.NoError(t, err)

	return path
}

func TestCommand(t *testing.T) {
	tests := []struct {
		name            string
		argv            []string
		expectedCommand string
	}{
		{name: "ResolvedBinary_Based", argv: []string{"/usr/bin/echo", "cast"}, expectedCommand: "echo cast"},
		{name: "BareBinary_Unchanged", argv: []string{"echo"}, expectedCommand: "echo"},
		{name: "NoArgv_Empty", argv: nil, expectedCommand: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCommand, Invocation{Argv: tt.argv}.Command())
		})
	}
}

func TestRunModes(t *testing.T) {
	tests := []struct {
		name           string
		mode           Mode
		expectedSink   string
		expectedOutput string
	}{
		{name: "Stream_SinkAndOutput", mode: Stream, expectedSink: "cast\n", expectedOutput: ""},
		{name: "Capture_OutputOnly", mode: Capture, expectedSink: "", expectedOutput: "cast\n"},
		{name: "Quiet_Neither", mode: Quiet, expectedSink: "", expectedOutput: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &bytes.Buffer{}

			result, err := Invoke(context.Background(), NewSettings(sink), Invocation{
				Argv: []string{binary(t, "echo"), "cast"},
				Mode: tt.mode,
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSink, sink.String())
			assert.Equal(t, tt.expectedOutput, string(result.Output))
		})
	}
}

func TestRunFailureCarriesTheTail(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
	}{
		{name: "Stream_Tailed", mode: Stream},
		{name: "Capture_Tailed", mode: Capture},
		{name: "Quiet_Tailed", mode: Quiet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Invoke(context.Background(), NewSettings(&bytes.Buffer{}), Invocation{
				Argv: []string{binary(t, "sh"), "-c", "echo the tool said no >&2; exit 3"},
				Mode: tt.mode,
			})

			require.Error(t, err)

			exception := foundryerrors.ExceptionOf(err)
			assert.Equal(t, "failed to run sh -c echo the tool said no >&2; exit 3: the tool said no\n", exception.Message)

			require.NotNil(t, exception.Cause)
			assert.Equal(t, "exit status 3", exception.Cause.Message)
		})
	}
}

// Stdin is never attached, so a tool that would prompt fails instead of
// hanging on a CI tooler with nothing to answer it.
func TestRunLeavesStdinUnattached(t *testing.T) {
	result, err := Invoke(context.Background(), Settings{}, Invocation{
		Argv: []string{binary(t, "sh"), "-c", "cat"},
		Mode: Capture,
	})

	assert.NoError(t, err)
	assert.Empty(t, string(result.Output))
}

func TestRunIgnoresACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Invoke(ctx, Settings{}, Invocation{Argv: []string{binary(t, "echo"), "cast"}, Mode: Capture})

	assert.NoError(t, err)
	assert.Equal(t, "cast\n", string(result.Output))
}

func TestRunValidatesTheInvocation(t *testing.T) {
	tests := []struct {
		name       string
		invocation Invocation
	}{
		{name: "NoMode_Invalid", invocation: Invocation{Argv: []string{binary(t, "echo"), "cast"}}},
		{name: "NoArgv_Invalid", invocation: Invocation{Mode: Capture}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Invoke(context.Background(), Settings{}, tt.invocation)

			require.Error(t, err)
			assert.Contains(t, foundryerrors.ExceptionOf(err).Message, "failed to build invocation")
		})
	}
}
