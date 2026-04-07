package segmentledger

import (
	"context"
	"fmt"
	"runtime"

	"github.com/denisbrodbeck/machineid"
	segment "github.com/segmentio/analytics-go/v3"
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

func (p *provider) Track(_ context.Context, event string, properties map[string]any) {
	if properties == nil {
		properties = make(map[string]any)
	}

	properties["os"] = runtime.GOOS
	properties["arch"] = runtime.GOARCH
	properties["foundry_version"] = version.Info.Version()

	props := segment.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	_ = p.client.Enqueue(segment.Track{
		AnonymousId: getDistinctID(),
		Event:       fmt.Sprintf("foundryctl: %s", event),
		Properties:  props,
	})
}

func (p *provider) Close() error {
	return p.client.Close()
}

// getDistinctID returns a hashed machine ID for anonymous attribution.
// The ID is stable across sessions and not reversible to the original machine ID.
func getDistinctID() string {
	id, err := machineid.ProtectedID("foundryctl")
	if err != nil {
		return "unknown"
	}
	if len(id) > 32 {
		return id[:32]
	}
	return id
}
