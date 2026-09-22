package dockercomposecasting

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
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
