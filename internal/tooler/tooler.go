// Package tooler is foundry's tool-interaction layer: one package per tool
// under internal/tooler, shared by every casting whose flavor speaks that
// tool. A tooler is casting-, kind- and set-blind and holds no deployment
// state; a casting states everything about a call in what it passes to the
// verb.
//
// The principles:
//
//   - Relay, never interpret: the tool's words are the diagnostic; foundry
//     never parses tool output into meaning.
//   - Invoker fidelity: invoke exactly as the user would at their shell,
//     never weakening a tool's own guard.
//   - Stdout is foundry's contract: only foundry-shaped documents on stdout;
//     every diagnostic byte goes to stderr.
//   - Mode belongs to the verb: how output is consumed is decided here,
//     never by a caller.
//   - The membrane: what is invariant across callers is tooler-side; what
//     varies per casting is casting-side.
//   - Gauge, never provision: a tooler proves a tool is usable and hints at
//     the fix; it never installs one.
//   - The world is the record: no invocation state; the record is the lock,
//     the pours, and the tool's own record plus ownership stamps.
//
// Verbs are the tool's real operations, 1:1, with ceremony absorbed; the
// tool's own composites are allowed, invented ones are not. Each belongs to
// one class:
//
//   - Mutation: Stream, stamps ownership on creation and verifies it on
//     destruction, takes approval where the tool demands it, re-entrant.
//   - Query: Capture, read-only, never approved.
//   - Connection: Stream, writes the user's own connection surface
//     tool-natively, stamps nothing.
//
// Options carry target and context names only, never credentials: connection
// is ambient in the environment, approval in the context (WithApproval,
// stamped once per command at the entrypoint). The Release is the deployable
// unit: domain.Release (name, owner) completed by each tooler's typed body.
// Mutation verbs act on it, Query verbs read by it, Connection verbs take
// none; it is constructed, never stored.
//
// A verb refuses in foundry's words before invoking, on a statement check or
// a world check, only where the tool would otherwise lie or misfire; never
// to duplicate validation the tool performs loudly. A check that needs
// casting knowledge is orchestration and belongs in Cast. Failures classify
// as reach (the tool is absent or unusable; TypeNotFound with the fix hint),
// precondition (the tooler refused; the tool never ran) or conversation (the
// tool ran and said no; TypeInternal, naming the verb, carrying the tool's
// last words; interrupted runs land here too).
//
// Every tooler answers three checks for its mutating verbs: reach (Gauge,
// connection-independent), context (inside the verb, connection-dependent)
// and ownership (stamp on creation, verify on destruction, list-by on
// demand). One that cannot does not land.
//
// Invoke holds the shared mechanics: argv only and never a shell, environment
// untouched, no chdir, tail-bounded output; castings never call it. Foundry
// forwards no signals: the kernel delivers the user's interrupt to the
// exec'd tool directly, foundry survives its own copy and keeps reading the
// streams, and Invoke ignores the context so a cancellation can never become a
// second delivery. SDK verbs are the other side: the kernel cannot reach
// them, so they must honor the cancelled context. Foundry puts no clock on
// a tool's work.
package tooler

import (
	"context"
	"os/exec"
)

type Tooler interface {
	Name() string

	Gauge(ctx context.Context) error
}

type approvalKey struct{}

func WithApproval(ctx context.Context) context.Context {
	return context.WithValue(ctx, approvalKey{}, true)
}

// Approved fails closed: a missing stamp refuses, never an unapproved run.
func Approved(ctx context.Context) bool {
	yes, _ := ctx.Value(approvalKey{}).(bool)

	return yes
}

// Resolve returns the absolute path of a tool on PATH.
func Resolve(name string) (string, error) {
	return exec.LookPath(name)
}
