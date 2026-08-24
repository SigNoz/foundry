package awstooler

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

type Options struct {
	ClusterName string

	Region string
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
		if aws, ok := t.(*Tooler); ok {
			return aws, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the aws tooler: it is not registered for this casting")
}

func (t *Tooler) Name() string {
	return "aws"
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.probe(ctx)

	return err
}

// UpdateKubeconfig writes the cluster's kubecontext and leaves it selected;
// the alias is the cluster name, not the ARN, so it stays derivable.
func (t *Tooler) UpdateKubeconfig(ctx context.Context, options Options) error {
	if options.ClusterName == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to run aws eks update-kubeconfig: no cluster is stated")
	}

	dialect, err := t.probe(ctx)
	if err != nil {
		return err
	}

	argv := append(slices.Clone(dialect.argv[1:]), "eks", "update-kubeconfig", "--name", options.ClusterName, "--alias", options.ClusterName)
	if options.Region != "" {
		argv = append(argv, "--region", options.Region)
	}

	t.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{dialect.argv[0]}, argv...), " ")))

	_, err = tooler.Invoke(ctx, tooler.Settings{Sink: t.sink}, tooler.Invocation{
		Verb: dialect.name + " eks update-kubeconfig",
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

	path, err := tooler.Resolve("aws")
	if err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find aws: install it from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html")
	}

	if _, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "aws --version",
		Path: path,
		Args: []string{"--version"},
		Mode: tooler.Capture,
	}); err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find aws: install it from https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html")
	}

	t.dialect = dialect{name: "aws", argv: []string{path}}

	return t.dialect, nil
}
