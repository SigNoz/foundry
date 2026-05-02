package segmentledger

import (
	"context"
	"fmt"
	"runtime"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/ledger"
	"github.com/signoz/foundry/internal/ledger/noopledger"
	"github.com/signoz/foundry/internal/version"
)

// provider implements ledger.Ledger using Segment.
type provider struct {
	client segment.Client
}

// New creates a new Segment ledger provider.
// Returns a noop provider if the write key is not set.
func New(config ledger.Config) ledger.Ledger {
	if config.Segment.Key == "" || config.Segment.Key == "<unset>" {
		return noopledger.New()
	}

	client, err := segment.NewWithConfig(config.Segment.Key, segment.Config{})
	if err != nil {
		return noopledger.New()
	}

	return &provider{
		client: client,
	}
}

func (p *provider) Track(_ context.Context, event domain.Event, properties domain.Properties) {
	properties = properties.
		Set("os", runtime.GOOS).
		Set("arch", runtime.GOARCH).
		Set("foundry_version", version.Info.Version())

	props := segment.NewProperties()
	for k, v := range properties.Map() {
		props.Set(k, v)
	}

	_ = p.client.Enqueue(segment.Track{
		AnonymousId: domain.MustNewDistinctID().String(),
		Event:       fmt.Sprintf("foundryctl: %s", event.String()),
		Properties:  props,
	})
}

func (p *provider) Close() error {
	return p.client.Close()
}
