package segmentledger

import (
	"context"
	"os"
	"runtime"

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

func (p *provider) Track(_ context.Context, properties map[string]any) {
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
		Event:       "foundryctl",
		Properties:  props,
	})
}

func (p *provider) Close() error {
	return p.client.Close()
}

// getDistinctID returns a stable anonymous identifier for the machine.
// It uses the hostname so there is no PII stored.
func getDistinctID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
