package tooler

import (
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

	tail := &errors.Tail{}
	captured := inv.Mode.wire(cmd, sink, tail)

	if err := cmd.Run(); err != nil {
		// The tool's last words are the reason.
		if words := tail.String(); words != "" {
			return Result{}, errors.Wrapf(err, errors.TypeInternal, "failed to run %s: %s", inv.Verb, words)
		}

		return Result{}, errors.Wrapf(err, errors.TypeInternal, "failed to run %s", inv.Verb)
	}

	if captured == nil {
		return Result{}, nil
	}

	return Result{Output: captured.Bytes()}, nil
}
