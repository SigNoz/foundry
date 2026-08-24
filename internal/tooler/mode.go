package tooler

import (
	"bytes"
	"io"
	"os/exec"

	"github.com/signoz/foundry/internal/errors"
)

// Mode is how a verb consumes its tool's output, decided by the verb and
// never by a caller. The behavior-carrying variant shape follows
// domain.Format.
type Mode struct {
	wire func(cmd *exec.Cmd, sink io.Writer, tail *errors.Tail) *bytes.Buffer
}

var (
	// The shared writer makes os/exec serialize the two streams into one.
	Stream = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *errors.Tail) *bytes.Buffer {
		out := io.MultiWriter(sink, tail)
		cmd.Stdout, cmd.Stderr = out, out

		return nil
	}}

	Capture = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *errors.Tail) *bytes.Buffer {
		buf := &bytes.Buffer{}
		cmd.Stdout, cmd.Stderr = buf, tail

		return buf
	}}

	Quiet = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, tail *errors.Tail) *bytes.Buffer {
		cmd.Stdout, cmd.Stderr = tail, tail

		return nil
	}}
)
