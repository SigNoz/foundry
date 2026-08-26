// Package terraformtooler speaks terraform.
package terraformtooler

import (
	"context"
	"log/slog"
	"slices"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

type Release struct {
	domain.Release

	Root string
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.Root == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no root is stated")
	}

	return nil
}

type Tooler struct {
	tooler.Tool

	// words is the resolved command prefix, memoized by command.
	words []string
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("terraform", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if terraform, ok := t.(*Tooler); ok {
			return terraform, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the terraform tooler: it is not registered for this casting")
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.command(ctx)

	return err
}

// Apply runs init first (idempotent) so a never-initialised root still applies.
func (t *Tooler) Apply(ctx context.Context, release Release) error {
	// terraform prompts before changing infra; foundry re-homes that to --yes.
	if !tooler.Approved(ctx) {
		return errors.Newf(errors.TypeInvalidInput, "failed to run terraform apply: no approval is stated; re-run with --yes")
	}

	if err := t.run(ctx, release, "init"); err != nil {
		return err
	}

	return t.run(ctx, release, "apply", "-auto-approve")
}

func (t *Tooler) Destroy(ctx context.Context, release Release) error {
	if !tooler.Approved(ctx) {
		return errors.Newf(errors.TypeInvalidInput, "failed to run terraform destroy: no approval is stated; re-run with --yes")
	}

	if err := t.run(ctx, release, "init"); err != nil {
		return err
	}

	return t.run(ctx, release, "destroy", "-auto-approve")
}

func (t *Tooler) run(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	words, err := t.command(ctx)
	if err != nil {
		return err
	}

	// -chdir tells terraform the root; foundry never changes its own cwd.
	argv := append(slices.Clone(words), "-chdir="+release.Root, verb)
	argv = append(argv, args...)

	inv := tooler.Invocation{Argv: argv, Mode: tooler.Stream}

	t.Logger.DebugContext(ctx, "running command", slog.String("command", inv.Command()))

	_, err = tooler.Invoke(ctx, t.Settings, inv)

	return err
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) command(ctx context.Context) ([]string, error) {
	if len(t.words) != 0 {
		return t.words, nil
	}

	path, err := tooler.Resolve("terraform")
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to find terraform: install it from https://developer.hashicorp.com/terraform/install")
	}

	if _, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
		Argv: []string{path, "version"},
		Mode: tooler.Capture,
	}); err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to find terraform: install it from https://developer.hashicorp.com/terraform/install")
	}

	t.words = []string{path}

	return t.words, nil
}
