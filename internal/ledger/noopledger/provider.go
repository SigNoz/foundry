package noopledger

import (
	"context"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/ledger"
)

// provider is a no-op implementation of ledger.Ledger.
type provider struct{}

// New creates a ledger that does nothing.
func New() ledger.Ledger {
	return &provider{}
}

func (p *provider) Track(_ context.Context, _ domain.Event, _ domain.Properties) {}
func (p *provider) Close() error                                                 { return nil }
