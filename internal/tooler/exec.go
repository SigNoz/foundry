package tooler

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// waitForOutput bounds the post-exit wait for stream pipes an orphaned
// grandchild can hold open forever.
const waitForOutput = 10 * time.Second

type Invocation struct {
	// Argv[0] is the resolved binary. Never a shell string: injection is
	// structurally impossible.
	Argv []string

	Mode Mode
}

// Everything here is tooler-constructed, so a gap is an internal fault, never
// user input.
func (inv Invocation) Validate() error {
	if len(inv.Argv) == 0 {
		return foundryerrors.Newf(foundryerrors.TypeInternal, "failed to build invocation: no command is stated")
	}

	if inv.Mode.wire == nil {
		return foundryerrors.Newf(foundryerrors.TypeInternal, "failed to build invocation for %s: no output mode is stated", inv.Command())
	}

	return nil
}

// Command renders the invocation the way the user would type it, dropping the
// resolved binary's directory.
func (inv Invocation) Command() string {
	if len(inv.Argv) == 0 {
		return ""
	}

	return strings.Join(append([]string{filepath.Base(inv.Argv[0])}, inv.Argv[1:]...), " ")
}

type Mode struct {
	wire func(cmd *exec.Cmd, sink io.Writer, tail *foundryerrors.Tail) *bytes.Buffer
}

var (
	// The shared writer makes os/exec serialize the two streams into one.
	Stream = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *foundryerrors.Tail) *bytes.Buffer {
		out := io.MultiWriter(sink, tail)
		cmd.Stdout, cmd.Stderr = out, out

		return nil
	}}

	Capture = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *foundryerrors.Tail) *bytes.Buffer {
		buf := &bytes.Buffer{}
		cmd.Stdout, cmd.Stderr = buf, tail

		return buf
	}}

	Quiet = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *foundryerrors.Tail) *bytes.Buffer {
		cmd.Stdout, cmd.Stderr = tail, tail

		return nil
	}}
)

type Result struct {
	// Output is filled in Capture mode alone.
	Output []byte
}

// Invoke owns every process spawn in foundry, and castings never reach it. A
// tool runs as the user would run it at their shell: same environment, same
// working directory, no clock on its work.
func Invoke(ctx context.Context, settings Settings, inv Invocation) (Result, error) {
	if err := inv.Validate(); err != nil {
		return Result{}, err
	}

	// ctx is unused: the kernel already delivered the interrupt directly.
	cmd := exec.Command(inv.Argv[0], inv.Argv[1:]...)

	cmd.WaitDelay = waitForOutput

	tail := &foundryerrors.Tail{}
	captured := inv.Mode.wire(cmd, settings.Sink(), tail)

	if err := cmd.Run(); err != nil {
		if words := tail.String(); words != "" {
			return Result{}, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run %s: %s", inv.Command(), words)
		}

		return Result{}, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run %s", inv.Command())
	}

	if captured == nil {
		return Result{}, nil
	}

	return Result{Output: captured.Bytes()}, nil
}

func Resolve(name string) (string, error) {
	return exec.LookPath(name)
}
