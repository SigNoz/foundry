package awsrunner

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/runner"
)

var _ runner.Runner = (*Runner)(nil)

// Options is what a casting states about one call. The runner declares the
// tool's vocabulary; each casting fills in what it needs, so one casting's
// knobs never reach another's calls and the runner carries no casting state.
type Options struct {
	// ClusterName names the EKS cluster whose kubecontext is written.
	ClusterName string

	// Region holding the cluster. Empty lets the CLI resolve its own.
	Region string

	// Stdout and Stderr carry the tool's own output. Nil uses the process
	// streams.
	Stdout io.Writer
	Stderr io.Writer
}

// Runner interacts with the aws CLI. Everything about a call arrives with it,
// so the runner holds nothing but how to reach the tool.
type Runner struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Runner {
	return &Runner{logger: logger}
}

// Lookup picks the aws runner out of the runners a casting receives. It lives
// here so the contract package never imports its implementations.
func Lookup(runners []runner.Runner) (*Runner, error) {
	for _, r := range runners {
		if aws, ok := r.(*Runner); ok {
			return aws, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the aws runner: it is not registered for this casting")
}

func (r *Runner) Name() string {
	return "aws"
}

func (r *Runner) Gauge(ctx context.Context) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return errors.Wrapf(err, errors.TypeNotFound, "failed to find aws: install it from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html")
	}

	return nil
}

// UpdateKubeconfig writes the cluster's kubecontext into the user's own
// kubeconfig and leaves it selected, which is the CLI's behavior and what a
// following apply relies on. The context is named after the cluster rather
// than its ARN, so the name stays derivable.
func (r *Runner) UpdateKubeconfig(ctx context.Context, options Options) error {
	if options.ClusterName == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to run aws eks update-kubeconfig: no cluster is stated")
	}

	return r.run(ctx, options, "eks", "update-kubeconfig", "--name", options.ClusterName, "--alias", options.ClusterName)
}

func (r *Runner) run(ctx context.Context, options Options, args ...string) error {
	if options.Region != "" {
		args = append(args, "--region", options.Region)
	}

	r.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{"aws"}, args...), " ")))

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if options.Stdout != nil {
		cmd.Stdout = options.Stdout
	}

	if options.Stderr != nil {
		cmd.Stderr = options.Stderr
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run aws %s", args[0])
	}

	return nil
}
