package update

// repository is the GitHub repository to query for releases, set via ldflags
// at build time. Empty in dev builds so go run / go build skip the check.
// Example: -ldflags "-X github.com/signoz/foundry/internal/update.repository=SigNoz/foundry".
var repository string = "<unset>"

// Config holds update notifier configuration.
type Config struct {
	Enabled bool
}

// NewConfig returns the default update notifier configuration.
func NewConfig() Config {
	return Config{
		Enabled: true,
	}
}
