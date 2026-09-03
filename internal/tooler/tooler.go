// Package tooler is how foundry speaks to the tools that carry out a
// deployment: docker compose, helm, terraform, systemctl, the Kubernetes and
// AWS APIs. One package per tool; castings drive toolers and toolers drive
// tools. A tooler knows nothing about castings or kinds, so everything a call
// depends on arrives in the verb's arguments.
//
// A tool is reached in one of two shapes. An exec tooler runs a binary through
// Invoke, which states the rules that spawning carries. An SDK tooler calls a
// client library in-process and honours the cancelled context itself, since no
// interrupt is delivered to it.
//
// Verbs are the tool's real operations, one to one. A Mutation acts on a
// Release: it validates, probes the tool, verifies the ownership labels
// stamped into the pours, then invokes. Those labels and the platform's own
// record are the only deployment state there is. A Query reads the world
// instead, and never waits on approval.
//
// Release, Tooler and Connection are the core. A Release is constructed per
// call and never stored, a Tooler holds only the tool's name and settings, and
// connection and approval are ambient rather than arguments.
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
