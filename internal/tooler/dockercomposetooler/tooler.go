// Package dockercomposetooler speaks docker compose.
package dockercomposetooler

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

// Release is the deployable unit the casting drafts for a compose project; the
// project compose keys on is the release name.
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
	return &Tooler{Tool: tooler.NewTool("docker compose", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if compose, ok := t.(*Tooler); ok {
			return compose, nil
		}
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to look up the compose tooler: it is not registered for this casting")
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.command(ctx)

	return err
}

// Up returns once the containers are started, not once they are healthy.
func (t *Tooler) Up(ctx context.Context, release Release) error {
	return t.run(ctx, release, "up", "-d")
}

// Down removes containers and networks; volumes stay (the data line).
func (t *Tooler) Down(ctx context.Context, release Release) error {
	return t.run(ctx, release, "down")
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

	argv := append(slices.Clone(words), "-f", release.File, verb)
	argv = append(argv, args...)

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
			"--filter", "label=com.docker.compose.project=" + release.Name,
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

	path, _ := tooler.Resolve("docker")

	if path != "" {
		_, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
			Argv: []string{path, "compose", "version"},
			Mode: tooler.Capture,
		})
		if err == nil {
			t.words = []string{path, "compose"}

			return t.words, nil
		}
	}

	if legacy, err := tooler.Resolve("docker-compose"); err == nil {
		t.words = []string{legacy}

		return t.words, nil
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to find docker compose: install the docker compose plugin or docker-compose")
}
