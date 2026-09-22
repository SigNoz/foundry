package dockerswarmcasting

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotEmptyAndValid(t *testing.T) {
	assert.NotEmpty(t, composeYAMLTemplate)

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, collectionagent.Default())

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())

	assert.NotEmpty(t, agentYAMLTemplate)

	buf.Reset()
	err = agentYAMLTemplate.Execute(buf, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())
}

func TestForgeUnsupportedKind(t *testing.T) {
	config := *collectionagent.Default()
	config.Spec.Collector.Kind = collectionagent.CollectorKindSidecar

	assert.Error(t, New(slog.New(slog.DiscardHandler)).Forge(t.Context(), config, pourer.New("collectionagent")))
}

// One agent per host reads that host's containers, so a replica count has
// nothing to scale.
func TestReplicas(t *testing.T) {
	for _, test := range []struct {
		name     string
		replicas *int
		pass     bool
	}{
		{"Unstated_Valid", nil, true},
		{"One_Valid", domain.NewIntPtr(1), true},
		{"Two_Invalid", domain.NewIntPtr(2), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas

			err := newDockerSwarmMoldingEnricher().EnrichStatus(t.Context(), v1alpha1.MoldingKindCollector, config)

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
