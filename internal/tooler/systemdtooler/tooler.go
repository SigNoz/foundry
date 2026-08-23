package systemdtooler

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

// Release is the deployable unit the casting drafts for a systemctl call.
type Release struct {
	domain.Release

	// Units are the *.service file paths enabled, started, and stopped.
	Units []string
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if len(r.Units) == 0 {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no units are stated")
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
		if systemd, ok := r.(*Tooler); ok {
			return systemd, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the systemd tooler: it is not registered for this casting")
}

func (r *Tooler) Name() string {
	return "systemctl"
}

func (r *Tooler) Gauge(ctx context.Context) error {
	_, err := r.probe(ctx)

	return err
}

// Up enables the units by path, reloads so systemd picks up the new files, then
// starts them; --no-block returns before long-running services converge.
func (r *Tooler) Up(ctx context.Context, release Release) error {
	if err := r.mutate(ctx, release, "enable", release.Units...); err != nil {
		return err
	}

	if err := r.mutate(ctx, release, "daemon-reload"); err != nil {
		return err
	}

	return r.mutate(ctx, release, "start", append([]string{"--no-block"}, names(release.Units)...)...)
}

// Down stops and disables the units; the unit files and provisioned state stay.
func (r *Tooler) Down(ctx context.Context, release Release) error {
	if err := r.mutate(ctx, release, "stop", names(release.Units)...); err != nil {
		return err
	}

	if err := r.mutate(ctx, release, "disable", names(release.Units)...); err != nil {
		return err
	}

	return r.mutate(ctx, release, "daemon-reload")
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

// enable takes unit paths; start and stop take unit names.
func names(units []string) []string {
	out := make([]string, 0, len(units))
	for _, unit := range units {
		out = append(out, filepath.Base(unit))
	}

	return out
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (r *Tooler) probe(ctx context.Context) (dialect, error) {
	if len(r.dialect.argv) != 0 {
		return r.dialect, nil
	}

	path, err := tooler.Resolve("systemctl")
	if err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find systemctl: this host is not running systemd")
	}

	if _, err := tooler.Invoke(ctx, tooler.Settings{}, tooler.Invocation{
		Verb: "systemctl --version",
		Path: path,
		Args: []string{"--version"},
		Mode: tooler.Capture,
	}); err != nil {
		return dialect{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find systemctl: this host is not running systemd")
	}

	r.dialect = dialect{name: "systemctl", argv: []string{path}}

	return r.dialect, nil
}
