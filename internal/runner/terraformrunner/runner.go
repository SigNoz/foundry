package terraformrunner

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/runner"
)

var _ runner.Runner = (*Runner)(nil)

// Options is what a casting states about one call. The runner declares the
// tool's vocabulary; each casting fills in what it needs, so one casting's
// knobs never reach another's calls and the runner carries no casting state.
type Options struct {
	// Root is the directory holding the terraform files to run against.
	Root string

	// Stdout and Stderr carry the tool's own output. Nil uses the process
	// streams.
	Stdout io.Writer
	Stderr io.Writer
}

// Runner interacts with terraform. Everything about a call arrives with it, so
// the runner holds nothing but how to reach the tool.
type Runner struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Runner {
	return &Runner{logger: logger}
}

// Lookup picks the terraform runner out of the runners a casting receives. It
// lives here so the contract package never imports its implementations.
func Lookup(runners []runner.Runner) (*Runner, error) {
	for _, r := range runners {
		if terraform, ok := r.(*Runner); ok {
			return terraform, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the terraform runner: it is not registered for this casting")
}

func (r *Runner) Name() string {
	return "terraform"
}

func (r *Runner) Gauge(ctx context.Context) error {
	if _, err := exec.LookPath("terraform"); err != nil {
		return errors.Wrapf(err, errors.TypeNotFound, "failed to find terraform: install it from https://developer.hashicorp.com/terraform/install")
	}

	return nil
}

// Apply converges the root. Init runs first: it is idempotent, and a root that
// was forged but never initialised cannot apply.
func (r *Runner) Apply(ctx context.Context, options Options) error {
	if err := r.run(ctx, options, "init"); err != nil {
		return err
	}

	return r.run(ctx, options, "apply", "-auto-approve")
}

// Destroy removes what the root's state records. Whatever the root holds goes
// with it, so a casting states the root deliberately.
func (r *Runner) Destroy(ctx context.Context, options Options) error {
	if err := r.run(ctx, options, "init"); err != nil {
		return err
	}

	return r.run(ctx, options, "destroy", "-auto-approve")
}

func (r *Runner) run(ctx context.Context, options Options, verb string, args ...string) error {
	if options.Root == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to run terraform: no root is stated")
	}

	// An unforged root is not an empty deployment: terraform reports an empty
	// directory as nothing to destroy and exits clean, which would let uncast
	// claim it removed a substrate it never saw.
	if !hasTerraformRoot(options.Root) {
		return errors.Newf(errors.TypeNotFound, "failed to run terraform: %q holds no terraform files; run forge first", options.Root)
	}

	// terraform is told the root with -chdir rather than the process working
	// directory, so a casting never has to move foundry to run it.
	full := append([]string{"-chdir=" + options.Root, verb}, args...)

	r.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{"terraform"}, full...), " ")))

	cmd := exec.CommandContext(ctx, "terraform", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if options.Stdout != nil {
		cmd.Stdout = options.Stdout
	}

	if options.Stderr != nil {
		cmd.Stderr = options.Stderr
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run terraform %s", verb)
	}

	return nil
}

// hasTerraformRoot reports whether the directory holds anything terraform
// would read as configuration.
func hasTerraformRoot(root string) bool {
	for _, pattern := range []string{"*.tf", "*.tf.json"} {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}
