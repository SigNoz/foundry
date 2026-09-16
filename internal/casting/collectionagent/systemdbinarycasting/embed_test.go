package systemdbinarycasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplates_Render(t *testing.T) {
	config := collectionagent.Default()
	config.Metadata.Name = "acme"
	config.Metadata.Annotations = map[string]string{collectionagent.CollectorAgentBinaryPath.Key: "/opt/otelcol/otelcol-contrib"}
	config.Spec.Collector.Spec.Env = map[string]string{
		"SIGNOZ_INGESTION_KEY":      "secret",
		"OTEL_RESOURCE_ATTRIBUTES":  "deployment.environment=production",
		"SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318",
	}

	testCases := []struct {
		name              string
		template          *domain.Template
		expectedContent   []string
		unexpectedContent []string
	}{
		{
			name:     "Unit_Valid",
			template: agentUnitTemplate,
			expectedContent: []string{
				"EnvironmentFile=/etc/acme-collector-agent/acme-collector-agent.conf\n",
				"ExecStart=/opt/otelcol/otelcol-contrib $OTELCOL_OPTIONS\n",
				"User=acme-collector-agent\n",
				"Group=acme-collector-agent\n",
				"AmbientCapabilities=CAP_DAC_READ_SEARCH\n",
				"SyslogIdentifier=acme-collector-agent\n",
			},
			unexpectedContent: []string{"DynamicUser", "StateDirectory", "SupplementaryGroups"},
		},
		{
			name:     "Sysusers_Valid",
			template: agentSysusersTemplate,
			expectedContent: []string{
				"u acme-collector-agent - \"SigNoz OpenTelemetry Collector\" - /usr/sbin/nologin\n",
			},
			unexpectedContent: []string{"m acme-collector-agent"},
		},
		{
			name:     "Environment_Valid",
			template: agentEnvironmentTemplate,
			expectedContent: []string{
				"OTELCOL_OPTIONS=--config=/etc/acme-collector-agent/agent.yaml\nOTEL_RESOURCE_ATTRIBUTES=deployment.environment=production\nSIGNOZ_INGESTION_ENDPOINT=http://signoz:4318\nSIGNOZ_INGESTION_KEY=secret\n",
			},
		},
		{
			name:     "Collector_Valid",
			template: agentYAMLTemplate,
			expectedContent: []string{
				"receivers:\n",
				"  hostmetrics:\n",
				"  filelog/varlog:\n",
				"      - /var/log/lastlog\n",
				"      receivers: [filelog/varlog]\n",
			},
			unexpectedContent: []string{"journald"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NotEmpty(t, testCase.template)

			buf := bytes.NewBuffer(nil)
			err := testCase.template.Execute(buf, config)

			require.NoError(t, err)
			for _, expected := range testCase.expectedContent {
				assert.Contains(t, buf.String(), expected)
			}

			for _, unexpected := range testCase.unexpectedContent {
				assert.NotContains(t, buf.String(), unexpected)
			}
		})
	}
}

func TestEnricherConfigTemplate_Render(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	require.NoError(t, agentYAMLTemplate.Execute(buf, nil))

	var parsed map[string]any
	require.NoError(t, domain.UnmarshalYAML(buf.Bytes(), &parsed))
	assert.Contains(t, parsed, "service")

	scrapers := parsed["receivers"].(map[string]any)["hostmetrics"].(map[string]any)["scrapers"].(map[string]any)
	assert.NotContains(t, scrapers, "nfs")

	process := scrapers["process"].(map[string]any)
	for _, flag := range []string{"mute_process_name_error", "mute_process_exe_error", "mute_process_io_error", "mute_process_user_error"} {
		assert.Contains(t, process, flag)
	}

	receivers := parsed["receivers"].(map[string]any)
	assert.NotContains(t, receivers, "journald")
	assert.Contains(t, receivers, "filelog/varlog")
	assert.Contains(t, receivers["filelog/varlog"].(map[string]any)["exclude"], "/var/log/lastlog")

	logs := parsed["service"].(map[string]any)["pipelines"].(map[string]any)["logs"].(map[string]any)
	assert.Equal(t, []any{"filelog/varlog"}, logs["receivers"])

	detectors := parsed["processors"].(map[string]any)["resourcedetection"].(map[string]any)["detectors"].([]any)
	assert.Equal(t, []any{"env", "ec2", "gcp", "azure", "ecs", "system"}, detectors)
}
