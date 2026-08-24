package tooler

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/signoz/foundry/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// binary resolves a tool every unix host has, so the tests prove the wiring
// against a real conversation rather than a fake.
func binary(t *testing.T, name string) string {
	t.Helper()

	path, err := exec.LookPath(name)
	require.NoError(t, err)

	return path
}

// The mode is what a verb chooses, and it decides where the tool's words go:
// to the user, to the caller, or nowhere. In every mode the tail keeps them.
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

			result, err := Invoke(context.Background(), Settings{Sink: sink}, Invocation{
				Verb: "echo cast",
				Path: binary(t, "echo"),
				Args: []string{"cast"},
				Mode: tt.mode,
			})

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSink, sink.String())
			assert.Equal(t, tt.expectedOutput, string(result.Output))
		})
	}
}

// A tool that says no is a conversation failure: the verb names it and the
// tool's own last words are the reason, whatever the mode streamed.
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
			_, err := Invoke(context.Background(), Settings{Sink: &bytes.Buffer{}}, Invocation{
				Verb: "sh -c",
				Path: binary(t, "sh"),
				Args: []string{"-c", "echo the tool said no >&2; exit 3"},
				Mode: tt.mode,
			})

			require.Error(t, err)

			exception := errors.ExceptionOf(err)
			assert.Equal(t, "failed to run sh -c: the tool said no\n", exception.Message)

			require.NotNil(t, exception.Cause)
			assert.Equal(t, "exit status 3", exception.Cause.Message)
		})
	}
}

// Stdin is never attached, so a tool that would prompt fails instead of
// hanging on a CI tooler with nothing to answer it.
func TestRunLeavesStdinUnattached(t *testing.T) {
	result, err := Invoke(context.Background(), Settings{}, Invocation{
		Verb: "sh -c cat",
		Path: binary(t, "sh"),
		Args: []string{"-c", "cat"},
		Mode: Capture,
	})

	assert.NoError(t, err)
	assert.Empty(t, string(result.Output))
}

// A cancelled context never reaches an exec'd tool: its interrupt channel is
// the kernel's, and Run must not open a second one.
func TestRunIgnoresACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Invoke(ctx, Settings{}, Invocation{Verb: "echo cast", Path: binary(t, "echo"), Args: []string{"cast"}, Mode: Capture})

	assert.NoError(t, err)
	assert.Equal(t, "cast\n", string(result.Output))
}

// A malformed invocation is a tooler bug, not user input: an unset mode or an
// unresolved binary fails with a typed error before anything is spawned,
// never a panic.
func TestRunValidatesTheInvocation(t *testing.T) {
	tests := []struct {
		name       string
		invocation Invocation
	}{
		{name: "NoMode_Invalid", invocation: Invocation{Verb: "echo", Path: binary(t, "echo"), Args: []string{"cast"}}},
		{name: "NoBinary_Invalid", invocation: Invocation{Verb: "echo", Mode: Capture}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Invoke(context.Background(), Settings{}, tt.invocation)

			require.Error(t, err)
			assert.Contains(t, errors.ExceptionOf(err).Message, "failed to build invocation")
		})
	}
}
