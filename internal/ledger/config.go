package ledger

// key is the Segment write key, set via ldflags at build time.
// Example: -ldflags "-X github.com/signoz/foundry/internal/ledger.key=<key>".
var key string = "<unset>"

// Config holds ledger configuration.
type Config struct {
	Enabled bool
	Segment Segment
}

// Segment holds Segment-specific configuration.
type Segment struct {
	Key string
}

// NewConfig returns the default ledger configuration.
// The Segment write key is populated from the ldflags-injected value.
func NewConfig() Config {
	return Config{
		Enabled: true,
		Segment: Segment{
			Key: key,
		},
	}
}

// Provider returns the provider name based on the configuration.
func (c Config) Provider() string {
	if c.Enabled {
		return "segment"
	}
	return "noop"
}
