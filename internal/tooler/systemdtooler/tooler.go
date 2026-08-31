// Package systemdtooler speaks systemctl.
package systemdtooler

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

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

// Options names a unit to read without asserting foundry ownership over it:
// Cat may probe a unit systemd itself never loaded from foundry, such as one a
// package manager installed.
type Options struct {
	Unit string
}

func (o Options) Validate() error {
	if o.Unit == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate options: no unit is stated")
	}

	return nil
}

type Tooler struct {
	tooler.Tool

	// words is the resolved command prefix, memoized by command.
	words []string
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("systemctl", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if systemd, ok := t.(*Tooler); ok {
			return systemd, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the systemd tooler: it is not registered for this casting")
}

func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.command(ctx)

	return err
}

// Up starts units with --no-block: it returns before they converge.
func (t *Tooler) Up(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if _, err := t.command(ctx); err != nil {
		return err
	}

	if err := t.verify(ctx, release); err != nil {
		return err
	}

	if err := t.run(ctx, release, "enable", release.Units...); err != nil {
		return err
	}

	if err := t.run(ctx, release, "daemon-reload"); err != nil {
		return err
	}

	return t.run(ctx, release, "start", append([]string{"--no-block"}, names(release.Units)...)...)
}

// Down stops and disables the units; the unit files and provisioned state stay.
func (t *Tooler) Down(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if _, err := t.command(ctx); err != nil {
		return err
	}

	if err := t.verify(ctx, release); err != nil {
		return err
	}

	if err := t.run(ctx, release, "stop", names(release.Units)...); err != nil {
		return err
	}

	if err := t.run(ctx, release, "disable", names(release.Units)...); err != nil {
		return err
	}

	return t.run(ctx, release, "daemon-reload")
}

// Restart applies a rewritten config: start alone no-ops on a unit already
// running.
func (t *Tooler) Restart(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if _, err := t.command(ctx); err != nil {
		return err
	}

	if err := t.verify(ctx, release); err != nil {
		return err
	}

	return t.run(ctx, release, "restart", names(release.Units)...)
}

// Cat reads a unit's full definition, drop-ins included. A unit systemd has
// never loaded is TypeNotFound.
func (t *Tooler) Cat(ctx context.Context, options Options) (string, error) {
	if err := options.Validate(); err != nil {
		return "", err
	}

	words, err := t.command(ctx)
	if err != nil {
		return "", err
	}

	result, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
		Argv: append(slices.Clone(words), "cat", options.Unit),
		Mode: tooler.Capture,
	})
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeNotFound, "failed to cat unit %q: it is not installed", options.Unit)
	}

	return string(result.Output), nil
}

// Owners reads each unit's [X-Foundry] Owner= line; a unit not yet installed
// contributes nothing rather than failing the read.
func (t *Tooler) Owners(ctx context.Context, release Release) (domain.Ownership, error) {
	if err := release.Release.Validate(); err != nil {
		return domain.Ownership{}, err
	}

	var owners []domain.Owner
	for _, unit := range release.Units {
		cat, err := t.Cat(ctx, Options{Unit: filepath.Base(unit)})
		if err != nil {
			continue
		}

		owners = append(owners, parseOwner(cat))
	}

	return domain.NewOwnership(owners...), nil
}

func (t *Tooler) verify(ctx context.Context, release Release) error {
	return tooler.Verify(ctx, t.Tool, release.Release, func(ctx context.Context) (domain.Ownership, error) {
		return t.Owners(ctx, release)
	})
}

func (t *Tooler) run(ctx context.Context, release Release, verb string, args ...string) error {
	if err := release.Validate(); err != nil {
		return err
	}

	words, err := t.command(ctx)
	if err != nil {
		return err
	}

	argv := append(slices.Clone(words), verb)
	argv = append(argv, args...)

	inv := tooler.Invocation{Argv: argv, Mode: tooler.Stream}

	t.Logger.DebugContext(ctx, "running command", slog.String("command", inv.Command()))

	_, err = tooler.Invoke(ctx, t.Settings, inv)

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

// parseOwner reads the Owner= line under a [X-Foundry] section from a unit's
// cat output; a unit with no such section returns a zero Owner.
func parseOwner(cat string) domain.Owner {
	inFoundrySection := false
	for line := range strings.SplitSeq(cat, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			inFoundrySection = trimmed == "[X-Foundry]"
			continue
		}

		if !inFoundrySection {
			continue
		}

		if owner, ok := strings.CutPrefix(trimmed, "Owner="); ok {
			return domain.ParseOwner(owner)
		}
	}

	return domain.Owner{}
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) command(ctx context.Context) ([]string, error) {
	if len(t.words) != 0 {
		return t.words, nil
	}

	path, err := tooler.Resolve("systemctl")
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to find systemctl: this host is not running systemd")
	}

	if _, err := tooler.Invoke(ctx, t.Settings, tooler.Invocation{
		Argv: []string{path, "--version"},
		Mode: tooler.Capture,
	}); err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to find systemctl: this host is not running systemd")
	}

	t.words = []string{path}

	return t.words, nil
}
