package kubectltooler

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

// Release is the deployable unit the casting drafts for a kubectl apply.
type Release struct {
	domain.Release

	// Dir is the kustomize root applied and deleted with -k.
	Dir string

	// URLs are manifests applied with -f before the kustomize and never
	// deleted: cluster-shared prerequisites like CRDs.
	URLs []string
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.Dir == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no directory is stated")
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
	for _, r := range toolers {
		if kubectl, ok := r.(*Tooler); ok {
			return kubectl, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the kubectl tooler: it is not registered for this casting")
}

func (r *Tooler) Name() string {
	return "kubectl"
}

func (r *Tooler) Gauge(ctx context.Context) error {
	_, err := r.probe(ctx)

	return err
}

// Apply lays the URL prerequisites down first, so the kustomize resources that
// depend on them resolve.
func (r *Tooler) Apply(ctx context.Context, release Release) error {
	if len(release.URLs) > 0 {
		args := make([]string, 0, len(release.URLs)*2)
		for _, url := range release.URLs {
			args = append(args, "-f", url)
		}

		if err := r.mutate(ctx, release, "apply", args...); err != nil {
			return err
		}
	}

	return r.mutate(ctx, release, "apply", "-k", release.Dir)
}

// Delete removes only the kustomize; the URL prerequisites are cluster-shared
// and stay.
func (r *Tooler) Delete(ctx context.Context, release Release) error {
	return r.mutate(ctx, release, "delete", "-k", release.Dir)
}

func (r *Tooler) mutate(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	dialect, err := r.probe(ctx)
	if err != nil {
		return err
	}

	argv := append(slices.Clone(dialect.argv[1:]), verb)
	argv = append(argv, args...)

	r.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{dialect.argv[0]}, argv...), " ")))

	_, err = tooler.Invoke(ctx, tooler.Settings{Sink: r.sink}, tooler.Invocation{
		Verb: dialect.name + " " + verb,
		Path: dialect.argv[0],
		Args: argv,
		Mode: tooler.Stream,
	})

	return err
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (r *Tooler) probe(ctx context.Context) (dialect, error) {
	if len(r.dialect.argv) != 0 {
		return r.dialect, nil
	}

	path, err := tooler.Resolve("kubectl")
	if err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find kubectl: install it from https://kubernetes.io/docs/tasks/tools/")
	}

	if _, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "kubectl version --client",
		Path: path,
		Args: []string{"version", "--client"},
		Mode: tooler.Capture,
	}); err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find kubectl: install it from https://kubernetes.io/docs/tasks/tools/")
	}

	r.dialect = dialect{name: "kubectl", argv: []string{path}}

	return r.dialect, nil
}
