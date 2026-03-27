package segmentledger

import (
	"context"
	"log/slog"
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
	logger *slog.Logger
}

// New creates a new Segment ledger provider.
// Returns a noop provider if the write key is not set.
func New(logger *slog.Logger, config ledger.Config) ledger.Ledger {
	if config.Segment.Key == "" || config.Segment.Key == "<unset>" {
		logger.Debug("ledger: segment write key not set, using noop provider")
		return noopledger.New()
	}

	client, err := segment.NewWithConfig(config.Segment.Key, segment.Config{
		Logger: &segmentLogger{logger: logger},
	})
	if err != nil {
		logger.Warn("ledger: failed to create segment client, using noop provider", slog.String("error", err.Error()))
		return noopledger.New()
	}

	return &provider{
		client: client,
		logger: logger,
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

	err := p.client.Enqueue(segment.Track{
		AnonymousId: getDistinctID(),
		Event:       "foundryctl",
		Properties:  props,
	})
	if err != nil {
		p.logger.Warn("ledger: failed to enqueue event", slog.String("error", err.Error()))
	}
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

// segmentLogger adapts slog.Logger to the segment.Logger interface.
type segmentLogger struct {
	logger *slog.Logger
}

func (l *segmentLogger) Logf(format string, args ...any) {
	l.logger.Debug("ledger: segment", slog.String("message", format))
}

func (l *segmentLogger) Errorf(format string, args ...any) {
	l.logger.Warn("ledger: segment error", slog.String("message", format))
}
