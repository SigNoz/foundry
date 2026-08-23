package tooler

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/signoz/foundry/internal/errors"
)

// pipeDrain bounds the post-exit wait for the stream pipes, which an orphaned
// grandchild can otherwise hold open forever. It never bounds the tool's work.
const pipeDrain = 10 * time.Second

type Mode struct {
	wire func(cmd *exec.Cmd, sink io.Writer, t *tail) *bytes.Buffer
}

var (
	// The shared writer makes os/exec serialize the two streams into one.
	Stream = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, t *tail) *bytes.Buffer {
		out := io.MultiWriter(sink, t)
		cmd.Stdout, cmd.Stderr = out, out

		return nil
	}}

	Capture = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, t *tail) *bytes.Buffer {
		buf := &bytes.Buffer{}
		cmd.Stdout, cmd.Stderr = buf, t

		return buf
	}}

	Quiet = Mode{wire: func(cmd *exec.Cmd, sink io.Writer, t *tail) *bytes.Buffer {
		cmd.Stdout, cmd.Stderr = t, t

		return nil
	}}
)

// Invocation is one command foundry runs against a tool, in its own terms.
type Invocation struct {
	Verb string

	Path string

	// Never a shell string: injection is structurally impossible.
	Args []string

	Mode Mode
}

// Everything here is tooler-constructed, so a gap is an internal fault, never
// user input.
func (inv Invocation) Validate() error {
	if inv.Path == "" {
		return errors.Newf(errors.TypeInternal, "failed to build invocation for %s: no binary resolved", inv.Verb)
	}

	if inv.Mode.wire == nil {
		return errors.Newf(errors.TypeInternal, "failed to build invocation for %s: no output mode stated", inv.Verb)
	}

	return nil
}

type Settings struct {
	// Sink is where streamed output goes; nil means stderr, and only tests set it.
	Sink io.Writer
}

type Result struct {
	// Output is filled in Capture mode alone.
	Output []byte
}

func Invoke(ctx context.Context, settings Settings, inv Invocation) (Result, error) {
	if err := inv.Validate(); err != nil {
		return Result{}, err
	}

	// No context: the tool's interrupt is the kernel's group delivery, already
	// made, and a cancelled context must not become a second one.
	cmd := exec.Command(inv.Path, inv.Args...)

	cmd.WaitDelay = pipeDrain

	sink := settings.Sink
	if sink == nil {
		sink = os.Stderr
	}

	t := &tail{}
	captured := inv.Mode.wire(cmd, sink, t)

	if err := cmd.Run(); err != nil {
		return Result{}, errors.Wrapf(err, errors.TypeInternal, "failed to run %s", inv.Verb).WithOutput(t.String())
	}

	if captured == nil {
		return Result{}, nil
	}

	return Result{Output: captured.Bytes()}, nil
}
