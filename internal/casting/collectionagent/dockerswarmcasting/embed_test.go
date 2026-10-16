package dockerswarmcasting

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

	config := collectionagent.Default()
	content := "receivers: {}\n"
	config.Spec.Collector.Spec.Config.Set(collectionagent.CollectorKindAgent.ConfigKey(), []byte(content))

	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())

	compose, err := domain.NewYAMLMaterial(buf.Bytes(), "compose.yaml")
	require.NoError(t, err)

	driver, err := compose.GetBytes("services.collector-agent.logging.driver")
	require.NoError(t, err)

	assert.Equal(t, "local", string(driver))

	hash := sha256.Sum256([]byte(content))

	name, err := compose.GetBytes("configs.collector-agent.name")
	require.NoError(t, err)

	assert.Equal(t, "signoz-collector-agent-"+hex.EncodeToString(hash[:])[:10], string(name))

	assert.NotEmpty(t, agentYAMLTemplate)

	buf.Reset()
	err = agentYAMLTemplate.Execute(buf, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, buf.String())

	agent, err := domain.NewYAMLMaterial(buf.Bytes(), "agent.yaml")
	require.NoError(t, err)

	override, err := agent.GetBytes("processors.resourcedetection.override")
	require.NoError(t, err)

	assert.Equal(t, "false", string(override))
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
		disabled bool
		pass     bool
	}{
		{"Unstated_Valid", nil, false, true},
		{"One_Valid", domain.NewIntPtr(1), false, true},
		{"Two_Invalid", domain.NewIntPtr(2), false, false},
		{"Disabled_Invalid", nil, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas
			config.Spec.Collector.Spec.Enabled = v1alpha1.BoolPtr(!test.disabled)

			err := newDockerSwarmMoldingEnricher().EnrichStatus(t.Context(), v1alpha1.MoldingKindCollector, config)

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
