package terraformtooler

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"

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

type dialect struct {
	name string
	argv []string
}

type Tooler struct {
	logger *slog.Logger

	dialect dialect

	// sink takes streamed output; nil is stderr. Only tests set it.
	sink io.Writer
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{logger: logger}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if terraform, ok := t.(*Tooler); ok {
			return terraform, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the terraform tooler: it is not registered for this casting")
}

func (t *Tooler) Name() string {
	return "terraform"
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.probe(ctx)

	return err
}

// Apply runs init first (idempotent) so a never-initialised root still applies.
func (t *Tooler) Apply(ctx context.Context, release Release) error {
	// terraform prompts before changing infra; foundry re-homes that to --yes.
	if !tooler.Approved(ctx) {
		return errors.Newf(errors.TypeInvalidInput, "failed to run terraform apply: no approval is stated; re-run with --yes")
	}

	if err := t.mutate(ctx, release, "init"); err != nil {
		return err
	}

	return t.mutate(ctx, release, "apply", "-auto-approve")
}

func (t *Tooler) Destroy(ctx context.Context, release Release) error {
	if !tooler.Approved(ctx) {
		return errors.Newf(errors.TypeInvalidInput, "failed to run terraform destroy: no approval is stated; re-run with --yes")
	}

	if err := t.mutate(ctx, release, "init"); err != nil {
		return err
	}

	return t.mutate(ctx, release, "destroy", "-auto-approve")
}

func (t *Tooler) mutate(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	dialect, err := t.probe(ctx)
	if err != nil {
		return err
	}

	// -chdir tells terraform the root; foundry never changes its own cwd.
	argv := append(slices.Clone(dialect.argv[1:]), "-chdir="+release.Root, verb)
	argv = append(argv, args...)

	t.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{dialect.argv[0]}, argv...), " ")))

	_, err = tooler.Invoke(ctx, tooler.Settings{Sink: t.sink}, tooler.Invocation{
		Verb: dialect.name + " " + verb,
		Path: dialect.argv[0],
		Args: argv,
		Mode: tooler.Stream,
	})

	return err
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) probe(ctx context.Context) (dialect, error) {
	if len(t.dialect.argv) != 0 {
		return t.dialect, nil
	}

	path, err := tooler.Resolve("terraform")
	if err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find terraform: install it from https://developer.hashicorp.com/terraform/install")
	}

	if _, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "terraform version",
		Path: path,
		Args: []string{"version"},
		Mode: tooler.Capture,
	}); err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find terraform: install it from https://developer.hashicorp.com/terraform/install")
	}

	t.dialect = dialect{name: "terraform", argv: []string{path}}

	return t.dialect, nil
}
