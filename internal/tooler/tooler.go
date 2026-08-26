// Package tooler is how foundry speaks to the tools that carry out a
// deployment: docker compose, helm, terraform, systemctl, the Kubernetes and
// AWS APIs. One package per tool; castings drive toolers and toolers drive
// tools. A tooler knows nothing about castings or kinds: everything a call
// depends on arrives in the verb's arguments.
//
// Toolers come in two shapes under one contract:
//
//   - exec toolers run a binary through Invoke
//   - SDK toolers (helm, kubernetes, aws) call a client library in-process
//
// Wherever the shapes differ, both sides are stated below.
//
// Every tool package has the same anatomy, so learning one teaches the rest
// (dockercomposetooler is the reference):
//
//   - Release embeds domain.Release (name, owner) and adds the fields its
//     tool needs (a compose file, a terraform root); Validate refuses a
//     missing field. It is constructed per call, never stored.
//   - Tooler embeds Tool (name, Logger, Settings) and implements the verbs.
//   - New constructs it; Lookup picks it out of a casting's registered set.
//   - Gauge proves the tool is usable: the binary resolves and answers its
//     version command, or the SDK resolves its configuration. Missing or
//     broken says what to install or fix; foundry never installs.
//
// Verbs are the tool's real operations, one-to-one; composites the tool
// ships are allowed, invented ones are not. Every verb is one of two
// classes:
//
//   - Mutation (Up, Apply, Down, Destroy) changes the world: it acts on a
//     Release, streams the tool's output live, and returns only an error.
//   - Query (Owners) reads the world: it captures output, answers with
//     values, and never waits on approval.
//
// A mutation body runs in a fixed order:
//
//  1. Validate: the Release refuses its own missing fields; verbs carry no
//     ad-hoc checks, and a check that needs casting knowledge belongs in
//     the casting's Cast, not here.
//  2. Probe: resolve the tool, the same proof Gauge makes, memoized.
//  3. Verify: read the ownership labels back from the platform and refuse a
//     name that belongs to someone else. The labels were stamped into the
//     pours; the platform's record plus those labels is the only deployment
//     state there is.
//  4. Invoke: run the tool; from here the words are its own.
//
// Errors are typed at the boundary they cross: the tool could not be found
// or would not answer is TypeNotFound, carrying the fix; the tooler refused
// before running it is TypeInvalidInput; the tool ran and said no is
// TypeInternal, naming the command and ending with the tool's last words
// (an interrupted run reads the same way). A verb refuses only where the
// tool would otherwise misfire or answer confusingly; checks the tool
// performs loudly are not duplicated.
//
// Invoke owns every process spawn; castings never call it. An Invocation is
// argv and a Mode: argv is a plain slice, never a shell string, so nothing
// can be injected, and the tool runs as the user would run it at their
// shell, same environment, same working directory, no weakened guards, no
// clock on its work. Mode pairs the verb class with its output: Stream
// shows it live (Mutation), Capture hands it back as Result (Query), Quiet
// keeps only enough to explain a failure. Whatever the mode, tool output
// lands on stderr (Settings.Sink); foundry's stdout carries nothing but
// foundry's documents, and foundry never reads meaning out of tool output.
//
// Signals and cancellation split by shape: the kernel delivers the user's
// interrupt to an exec'd tool directly, so Invoke deliberately ignores its
// context, while no delivery reaches an SDK call, so SDK verbs honor the
// cancelled context themselves.
//
// Connection and approval are both ambient. A tool reads the kubeconfig or
// credential chain it would read at the user's shell, unless a casting
// states an exact Connection (address, CA, token source); secrets move only
// through memory, never argv, files or logs. Approval is stamped once at
// the entrypoint by --yes (WithApproval), and a verb whose tool would
// prompt before changing infrastructure checks Approved and refuses without
// it.
package tooler

import (
	"context"
	"log/slog"
)

type Tooler interface {
	Name() string

	Gauge(ctx context.Context) error
}

// Tool is embedded by every tooler the way a Release embeds domain.Release.
type Tool struct {
	Logger *slog.Logger

	Settings Settings

	name string
}

func NewTool(name string, logger *slog.Logger) Tool {
	return Tool{Logger: logger, name: name}
}

func (t Tool) Name() string {
	return t.name
}
