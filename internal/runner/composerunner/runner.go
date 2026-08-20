package composerunner

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/runner"
)

var _ runner.Runner = (*Runner)(nil)

// ownerSeparator separates the label values docker returns for one container.
// No label foundry stamps carries it.
const ownerSeparator = "|"

// Options is what a casting states about one call. The runner declares the
// tool's vocabulary; each casting fills in what it needs, so one casting's
// knobs never reach another's calls and the runner carries no casting state.
type Options struct {
	// File is the compose file to run against.
	File string

	// Project names the compose project. Stating it together with Owner
	// guards the call.
	Project string

	// Owner guards the call: the runner refuses a project whose containers
	// report a different owner. What a casting stamps is what it claims.
	Owner domain.Owner

	// Stdout and Stderr carry the tool's own output. Nil uses the process
	// streams.
	Stdout io.Writer
	Stderr io.Writer
}

// command is how compose is invoked on this machine: the binary, plus the
// subcommand that precedes every call when the plugin is what is installed.
type command struct {
	name string
	args []string
}

// Runner interacts with docker compose. Everything about a call arrives with
// it, so the runner holds nothing but how to reach the tool.
type Runner struct {
	logger *slog.Logger

	// command is probed once: it is a property of the machine, not of a call.
	command command
}

func New(logger *slog.Logger) *Runner {
	return &Runner{logger: logger}
}

// Lookup picks the compose runner out of the runners a casting receives. It
// lives here so the contract package never imports its implementations.
func Lookup(runners []runner.Runner) (*Runner, error) {
	for _, r := range runners {
		if compose, ok := r.(*Runner); ok {
			return compose, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the compose runner: it is not registered for this casting")
}

func (r *Runner) Name() string {
	return "docker compose"
}

// Gauge resolves how compose is invoked on this machine: the docker compose
// plugin, or the legacy docker-compose binary.
func (r *Runner) Gauge(ctx context.Context) error {
	_, err := r.compose(ctx)

	return err
}

// Up converges the deployment and returns once the containers are started,
// not once they are healthy.
func (r *Runner) Up(ctx context.Context, options Options) error {
	return r.run(ctx, options, "up", "-d")
}

// Down removes the containers and networks the compose file created. Volumes
// stay: uncast never crosses the data line.
func (r *Runner) Down(ctx context.Context, options Options) error {
	return r.run(ctx, options, "down")
}

func (r *Runner) run(ctx context.Context, options Options, verb string, args ...string) error {
	if options.File == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to run docker compose: no compose file is stated")
	}

	if _, err := os.Stat(options.File); os.IsNotExist(err) {
		return errors.Newf(errors.TypeNotFound, "failed to run docker compose: no compose file at %q", options.File)
	}

	if err := r.checkOwner(ctx, options); err != nil {
		return err
	}

	command, err := r.compose(ctx)
	if err != nil {
		return err
	}

	full := append(append([]string{}, command.args...), "-f", options.File, verb)
	full = append(full, args...)

	r.logger.DebugContext(ctx, "running command", slog.String("command", strings.Join(append([]string{command.name}, full...), " ")))

	cmd := exec.CommandContext(ctx, command.name, full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if options.Stdout != nil {
		cmd.Stdout = options.Stdout
	}

	if options.Stderr != nil {
		cmd.Stderr = options.Stderr
	}

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run docker compose %s", verb)
	}

	return nil
}

// checkOwner refuses a project another owner already holds. Workloads that
// record no owner only warn: they are either a pre-label deployment or a
// foreign project. An unreadable engine skips the check rather than blocking
// the call, and a caller that states no owner is not guarded at all.
func (r *Runner) checkOwner(ctx context.Context, options Options) error {
	if options.Project == "" || len(options.Owner) == 0 {
		return nil
	}

	keys := slices.Sorted(maps.Keys(options.Owner))

	directives := make([]string, 0, len(keys))
	for _, key := range keys {
		directives = append(directives, `{{.Label "`+key+`"}}`)
	}

	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "label=com.docker.compose.project="+options.Project,
		"--format", strings.Join(directives, ownerSeparator)).Output()
	if err != nil {
		r.logger.WarnContext(ctx, "skipping the ownership check: could not read labels from docker", errors.LogAttr(err))

		return nil
	}

	ownership := domain.NewOwnership(owners(keys, string(out))...)

	if foreign, conflict := ownership.Foreign(options.Owner); conflict {
		return errors.Newf(errors.TypeInvalidInput, "failed to run docker compose: project %q already belongs to [%s] on this host, not [%s]: remove that deployment, or give this one a different name", options.Project, foreign, options.Owner)
	}

	if ownership.HasUnowned() {
		r.logger.WarnContext(ctx, "compose project has containers without ownership labels", slog.String("project", options.Project))
	}

	return nil
}

// owners reads back what the format asked for: one container per line, its
// label values in the order the keys were asked.
func owners(keys []string, out string) []domain.Owner {
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

// compose probes for the docker compose plugin, then for the legacy
// docker-compose binary, remembering the answer for later calls. The memo is
// unguarded on purpose: foundry runs single-threaded, so a lock here would
// claim a concurrency contract runners do not have.
func (r *Runner) compose(ctx context.Context) (command, error) {
	if r.command.name != "" {
		return r.command, nil
	}

	if _, err := exec.LookPath("docker"); err == nil {
		if err := exec.CommandContext(ctx, "docker", "compose", "version").Run(); err == nil {
			r.command = command{name: "docker", args: []string{"compose"}}

			return r.command, nil
		}
	}

	if _, err := exec.LookPath("docker-compose"); err == nil {
		r.command = command{name: "docker-compose"}

		return r.command, nil
	}

	return command{}, errors.Newf(errors.TypeNotFound, "failed to find docker compose: install the docker compose plugin or docker-compose")
}
