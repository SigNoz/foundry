// Package ledger provides anonymous usage tracking for foundryctl commands.
package ledger

import "context"

// Ledger is the interface for tracking CLI usage events.
type Ledger interface {
	// Track records a single foundryctl event with the given properties.
	Track(ctx context.Context, event string, properties map[string]any)

	// Close flushes any pending events and releases resources.
	Close() error
}
