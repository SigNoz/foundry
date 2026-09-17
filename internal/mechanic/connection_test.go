package mechanic

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/stretchr/testify/assert"
)

func TestResolveConnection(t *testing.T) {
	lock := func() *installation.Casting {
		c := installation.Default()
		c.Spec.Signoz.Status.Addresses.APIServer = []string{"signoz:8080"}
		c.Spec.TelemetryStore.Status.Addresses.TCP = []string{"tcp://ch-0:9000", "tcp://ch-1:9000"}
		c.Spec.MetaStore.Status.Addresses.DSN = []string{"postgres://meta:5432"}
		return c
	}

	tests := []struct {
		name      string
		machinery *installation.Casting
		overrides Overrides
		expected  Connection
	}{
		{
			name:      "LockFallback",
			machinery: lock(),
			expected: Connection{
				Signoz:     Endpoint{Value: "signoz:8080", Source: SourceLock},
				Clickhouse: Endpoint{Value: "tcp://ch-0:9000,tcp://ch-1:9000", Source: SourceLock},
				Metastore:  Endpoint{Value: "postgres://meta:5432", Source: SourceLock},
			},
		},
		{
			name:      "OverridesWin",
			machinery: lock(),
			overrides: Overrides{
				Signoz:        "signoz.example:443",
				ClickhouseDSN: "user:pass@ch.example:9000",
				MetastoreDSN:  "postgres://override:5432",
			},
			expected: Connection{
				Signoz:     Endpoint{Value: "signoz.example:443", Source: SourceOverride},
				Clickhouse: Endpoint{Value: "user:pass@ch.example:9000", Source: SourceOverride},
				Metastore:  Endpoint{Value: "postgres://override:5432", Source: SourceOverride},
			},
		},
		{
			name:      "PartialOverride",
			machinery: lock(),
			overrides: Overrides{ClickhouseDSN: "user:pass@ch.example:9000"},
			expected: Connection{
				Signoz:     Endpoint{Value: "signoz:8080", Source: SourceLock},
				Clickhouse: Endpoint{Value: "user:pass@ch.example:9000", Source: SourceOverride},
				Metastore:  Endpoint{Value: "postgres://meta:5432", Source: SourceLock},
			},
		},
		{
			name:      "NoLockNoOverride",
			machinery: installation.Default(),
			expected: Connection{
				Signoz:     Endpoint{Source: SourceUnset},
				Clickhouse: Endpoint{Source: SourceUnset},
				Metastore:  Endpoint{Source: SourceUnset},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := ResolveConnection(tt.machinery, tt.overrides)
			assert.Equal(t, tt.expected, conn)
		})
	}
}
