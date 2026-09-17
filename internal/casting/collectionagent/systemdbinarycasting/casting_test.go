package systemdbinarycasting

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForge(t *testing.T) {
	testCases := []struct {
		name string
		env  map[string]string
		pass bool
	}{
		{
			name: "SingleLineValues_Valid",
			env:  map[string]string{"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318", "OTEL_RESOURCE_ATTRIBUTES": "a=b,c=d %n"},
			pass: true,
		},
		{
			name: "MultiLineValue_Invalid",
			env:  map[string]string{"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318\nSIGNOZ_INGESTION_KEY=stolen"},
			pass: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Metadata.Name = "acme"
			config.Spec.Collector.Spec.Env = testCase.env

			p := pourer.New("collectionagent")
			err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *config, p)

			if !testCase.pass {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			materials, err := p.Pour()
			require.NoError(t, err)

			paths := make([]string, 0, len(materials))
			for _, material := range materials {
				paths = append(paths, material.Path())
			}

			assert.Contains(t, paths, "collectionagent/collector/agent/acme-collector-agent.service")
			assert.Contains(t, paths, "collectionagent/collector/agent/acme-collector-agent.conf")
			assert.Contains(t, paths, "collectionagent/collector/agent/acme-collector-agent.sysusers.conf")
		})
	}
}
