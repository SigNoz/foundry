// Package runner is the contract for foundry's tool-interaction layer: one
// package per tool under internal/runner, shared by every casting whose
// flavor speaks that tool.
//
// A runner is the interface between foundry and one tool. It is casting-,
// kind- and set-blind: no casting types, no ordering, no bindings. It holds
// nothing about any deployment either; a casting states everything about a
// call in the options it passes to the verb, so two castings of two Kinds can
// drive the same runner without either one's choices reaching the other.
package runner

import "context"

// Runner is the surface gauge needs. The tool's own operations live on each
// package's concrete type, which castings receive and use directly.
type Runner interface {
	Name() string

	// Gauge reports whether the tool can run here, in the same vocabulary
	// as tooler.Tooler. It needs nothing a casting knows, which is what
	// makes it callable at gauge time.
	Gauge(ctx context.Context) error
}
