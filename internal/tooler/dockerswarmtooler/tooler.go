// Package dockerswarmtooler speaks docker stack.
package dockerswarmtooler

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

// Release is the deployable unit the casting drafts for a swarm deploy; the
// stack compose keys on is the release name.
type Release struct {
	domain.Release

	File string
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.File == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to validate release: no compose file is stated")
	}

	return nil
}

type Tooler struct {
	tooler.Tool

	// words is the resolved command prefix, memoized by command.
	words []string
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("docker stack", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if swarm, ok := t.(*Tooler); ok {
			return swarm, nil
		}
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to look up the swarm tooler: it is not registered for this casting")
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.command(ctx)

	return err
}

// Up returns once the stack is accepted, not once its services converge.
func (t *Tooler) Up(ctx context.Context, release Release) error {
	return t.run(ctx, release, "deploy", "-d", "-c", release.File)
}

// Down removes the stack's services and networks; volumes stay (the data line).
func (t *Tooler) Down(ctx context.Context, release Release) error {
	return t.run(ctx, release, "rm")
}

func (t *Tooler) Owners(ctx context.Context, release Release) (domain.Ownership, error) {
	if err := release.Release.Validate(); err != nil {
		return domain.Ownership{}, err
	}

	return t.read(ctx, release)
}

func (t *Tooler) run(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	words, err := t.command(ctx)
	if err != nil {
		return err
	}

	if err := tooler.Verify(ctx, t.Tool, release.Release, func(ctx context.Context) (domain.Ownership, error) {
		return t.read(ctx, release)
	}); err != nil {
		return err
	}

	argv := append(slices.Clone(words), verb)
	argv = append(argv, args...)
	argv = append(argv, release.Name)

	inv := tooler.Invocation{Argv: argv, Mode: tooler.Stream}

	t.Logger.DebugContext(ctx, "running command", slog.String("command", inv.Command()))

	_, err = tooler.Invoke(ctx, t.Settings, inv)

	return err
}

func (t *Tooler) read(ctx context.Context, release Release) (domain.Ownership, error) {
	docker, err := tooler.Resolve("docker")
	if err != nil {
		return domain.Ownership{}, foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to run docker ps: docker is not available")
	}

	keys := slices.Sorted(maps.Keys(release.Owner))

	// Each container prints its owner in domain.Owner's String form, sorted, so
	// domain.ParseOwnership reads it straight back.
	directives := make([]string, 0, len(keys))
	for _, key := range keys {
		directives = append(directives, key+`={{.Label "`+key+`"}}`)
	}

	result, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
		Argv: []string{docker, "ps", "-a",
			"--filter", "label=com.docker.stack.namespace=" + release.Name,
			"--format", strings.Join(directives, ",")},
		Mode: tooler.Capture,
	})
	if err != nil {
		return domain.Ownership{}, err
	}

	return domain.ParseOwnership(string(result.Output)), nil
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) command(ctx context.Context) ([]string, error) {
	if len(t.words) != 0 {
		return t.words, nil
	}

	path, err := tooler.Resolve("docker")
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find docker: install it from https://docs.docker.com/engine/install/")
	}

	if _, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
		Argv: []string{path, "--version"},
		Mode: tooler.Capture,
	}); err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to find docker: install it from https://docs.docker.com/engine/install/")
	}

	t.words = []string{path, "stack"}

	return t.words, nil
}
