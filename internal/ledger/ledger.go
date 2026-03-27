package ledger

import "context"

// Ledger is the interface for tracking CLI usage events.
type Ledger interface {
	// Track sends a foundryctl event with the given properties.
	Track(ctx context.Context, properties map[string]any)

	// Close flushes any pending events and shuts down the client.
	Close() error
}
