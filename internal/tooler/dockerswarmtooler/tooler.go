package dockerswarmtooler

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
	for _, r := range toolers {
		if swarm, ok := r.(*Tooler); ok {
			return swarm, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the swarm tooler: it is not registered for this casting")
}

func (r *Tooler) Name() string {
	return "docker stack"
}

func (r *Tooler) Gauge(ctx context.Context) error {
	_, err := r.probe(ctx)

	return err
}

// Up returns once the stack is accepted, not once its services converge.
func (r *Tooler) Up(ctx context.Context, release Release) error {
	return r.mutate(ctx, release, "deploy", "-d", "-c", release.File)
}

// Down removes the stack's services and networks; volumes stay (the data line).
func (r *Tooler) Down(ctx context.Context, release Release) error {
	return r.mutate(ctx, release, "rm")
}

// Owners lists who holds the stack, read from its task containers' labels.
func (r *Tooler) Owners(ctx context.Context, release Release) (domain.Ownership, error) {
	if err := release.Release.Validate(); err != nil {
		return domain.Ownership{}, err
	}

	return r.list(ctx, release)
}

func (r *Tooler) mutate(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	dialect, err := r.probe(ctx)
	if err != nil {
		return err
	}

	if err := r.verify(ctx, release); err != nil {
		return err
	}

	argv := append(slices.Clone(dialect.argv[1:]), "stack", verb)
	argv = append(argv, args...)
	argv = append(argv, release.Name)

	r.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{dialect.argv[0]}, argv...), " ")))

	_, err = tooler.Invoke(ctx, tooler.Settings{Sink: r.sink}, tooler.Invocation{
		Verb: dialect.name + " stack " + verb,
		Path: dialect.argv[0],
		Args: argv,
		Mode: tooler.Stream,
	})

	return err
}

// An unreadable docker skips the ownership check rather than blocks; the verb
// itself then fails loudly, in the tool's own words.
func (r *Tooler) verify(ctx context.Context, release Release) error {
	ownership, err := r.list(ctx, release)
	if err != nil {
		r.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", errors.LogAttr(err))

		return nil
	}

	if foreign, conflict := ownership.Foreign(release.Owner); conflict {
		return errors.Newf(errors.TypeInvalidInput, "failed to run docker stack: stack %q already belongs to [%s], not [%s]: remove it, or deploy under a different name", release.Name, foreign, release.Owner)
	}

	if ownership.HasUnowned() {
		r.logger.WarnContext(ctx, "swarm stack has containers without ownership labels", slog.String("stack", release.Name))
	}

	return nil
}

func (r *Tooler) list(ctx context.Context, release Release) (domain.Ownership, error) {
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
			"--filter", "label=com.docker.stack.namespace=" + release.Name,
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
func (r *Tooler) probe(ctx context.Context) (dialect, error) {
	if len(r.dialect.argv) != 0 {
		return r.dialect, nil
	}

	path, err := tooler.Resolve("docker")
	if err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find docker: install it from https://docs.docker.com/engine/install/")
	}

	if _, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "docker --version",
		Path: path,
		Args: []string{"--version"},
		Mode: tooler.Capture,
	}); err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find docker: install it from https://docs.docker.com/engine/install/")
	}

	r.dialect = dialect{name: "docker", argv: []string{path}}

	return r.dialect, nil
}
