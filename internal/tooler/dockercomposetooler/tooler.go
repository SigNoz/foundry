package dockercomposetooler

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

// ownerSeparator joins the label values docker returns; no stamped label
// carries it.
const ownerSeparator = "|"

type Release struct {
	domain.Release

	File string
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.File == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no compose file is stated")
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
		if compose, ok := t.(*Tooler); ok {
			return compose, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the compose tooler: it is not registered for this casting")
}

func (t *Tooler) Name() string {
	return "docker compose"
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.probe(ctx)

	return err
}

// Up returns once the containers are started, not once they are healthy.
func (t *Tooler) Up(ctx context.Context, release Release) error {
	return t.mutate(ctx, release, "up", "-d")
}

// Down removes containers and networks; volumes stay (the data line).
func (t *Tooler) Down(ctx context.Context, release Release) error {
	return t.mutate(ctx, release, "down")
}

func (t *Tooler) Owners(ctx context.Context, release Release) (domain.Ownership, error) {
	if err := release.Release.Validate(); err != nil {
		return domain.Ownership{}, err
	}

	return t.list(ctx, release)
}

func (t *Tooler) mutate(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	dialect, err := t.probe(ctx)
	if err != nil {
		return err
	}

	if err := t.verify(ctx, release); err != nil {
		return err
	}

	argv := append(slices.Clone(dialect.argv[1:]), "-f", release.File, verb)
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

// An unreadable docker skips the ownership check rather than blocks; the verb
// itself then fails loudly, in the tool's own words.
func (t *Tooler) verify(ctx context.Context, release Release) error {
	ownership, err := t.list(ctx, release)
	if err != nil {
		t.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", errors.LogAttr(err))

		return nil
	}

	if foreign, conflict := ownership.Foreign(release.Owner); conflict {
		return errors.Newf(errors.TypeInvalidInput, "failed to run docker compose: project %q already belongs to [%s], not [%s]: remove it, or deploy under a different name", release.Name, foreign, release.Owner)
	}

	if ownership.HasUnowned() {
		t.logger.WarnContext(ctx, "compose project has containers without ownership labels", slog.String("project", release.Name))
	}

	return nil
}

func (t *Tooler) list(ctx context.Context, release Release) (domain.Ownership, error) {
	docker, err := tooler.Resolve("docker")
	if err != nil {
		return domain.Ownership{}, errors.Newf(errors.TypeNotFound, "failed to run docker ps: docker is not available")
	}

	keys := slices.Sorted(maps.Keys(release.Owner))

	directives := make([]string, 0, len(keys))
	for _, key := range keys {
		directives = append(directives, `{{.Label "`+key+`"}}`)
	}

	result, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "docker ps",
		Path: docker,
		Args: []string{"ps", "-a",
			"--filter", "label=com.docker.compose.project=" + release.Name,
			"--format", strings.Join(directives, ownerSeparator)},
		Mode: tooler.Capture,
	})
	if err != nil {
		return domain.Ownership{}, err
	}

	return domain.NewOwnership(parse(keys, string(result.Output))...), nil
}

// A container reporting nothing reads as unowned, not as an owner whose every
// value is empty.
func parse(keys []string, out string) []domain.Owner {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	parsed := make([]domain.Owner, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		values := strings.Split(line, ownerSeparator)

		owner := domain.Owner{}
		for i, key := range keys {
			if i < len(values) {
				owner[key] = values[i]
			}
		}

		parsed = append(parsed, owner)
	}

	return parsed
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) probe(ctx context.Context) (dialect, error) {
	if len(t.dialect.argv) != 0 {
		return t.dialect, nil
	}

	docker, _ := tooler.Resolve("docker")

	if docker != "" {
		_, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
			Verb: "docker compose version",
			Path: docker,
			Args: []string{"compose", "version"},
			Mode: tooler.Capture,
		})
		if err == nil {
			t.dialect = dialect{name: "docker compose", argv: []string{docker, "compose"}}

			return t.dialect, nil
		}
	}

	if legacy, err := tooler.Resolve("docker-compose"); err == nil {
		t.dialect = dialect{name: "docker-compose", argv: []string{legacy}}

		return t.dialect, nil
	}

	return dialect{}, errors.Newf(errors.TypeNotFound, "failed to find docker compose: install the docker compose plugin or docker-compose")
}
