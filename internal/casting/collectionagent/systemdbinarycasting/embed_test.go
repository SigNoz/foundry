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
				"User=signoz-collector-agent\n",
				"Group=signoz-collector-agent\n",
				"AmbientCapabilities=CAP_DAC_READ_SEARCH\n",
				"SyslogIdentifier=acme-collector-agent\n",
			},
			unexpectedContent: []string{"DynamicUser", "StateDirectory", "SupplementaryGroups"},
		},
		{
			name:     "Sysusers_Valid",
			template: agentSysusersTemplate,
			expectedContent: []string{
				"u signoz-collector-agent - \"SigNoz OpenTelemetry Collector\" - /usr/sbin/nologin\n",
			},
			unexpectedContent: []string{"acme"},
		},
		{
			name:     "Environment_Valid",
			template: agentEnvironmentTemplate,
			expectedContent: []string{
				"OTELCOL_OPTIONS=--config=/etc/acme-collector-agent/agent.yaml\nOTEL_RESOURCE_ATTRIBUTES=\"deployment.environment=production\"\nSIGNOZ_INGESTION_ENDPOINT=\"http://signoz:4318\"\nSIGNOZ_INGESTION_KEY=\"secret\"\n",
			},
		},
		{
			name:     "Collector_Valid",
			template: agentYAMLTemplate,
			expectedContent: []string{
				"receivers:\n",
				"  hostmetrics:\n",
				"  journald:\n",
				"      receivers: [journald]\n",
			},
			unexpectedContent: []string{"filelog"},
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

	devices := scrapers["disk"].(map[string]any)["exclude"].(map[string]any)["devices"]
	assert.Equal(t, []any{`^ram\d+$`, `^zram\d+$`, `^loop\d+$`, `^fd\d+$`, `^hd[a-z]\d+$`, `^sd[a-z]\d+$`, `^vd[a-z]\d+$`, `^xvd[a-z]\d+$`, `^nvme\d+n\d+p\d+$`}, devices)

	mountPoints := scrapers["filesystem"].(map[string]any)["exclude_mount_points"].(map[string]any)["mount_points"]
	assert.Equal(t, []any{"/dev/*", "/proc/*", "/sys/*", "/run/credentials/*", "/run/k3s/containerd/*", "/var/lib/docker/*", "/var/lib/containers/storage/*", "/var/lib/kubelet/*", "/snap/*"}, mountPoints)

	interfaces := scrapers["network"].(map[string]any)["exclude"].(map[string]any)["interfaces"]
	assert.Equal(t, []any{"^veth.*$", "^docker.*$", "^br-.*$", "^flannel.*$", "^cali.*$", "^cbr.*$", "^cni.*$", "^dummy.*$", "^tailscale.*$", "^lo$"}, interfaces)

	receivers := parsed["receivers"].(map[string]any)
	assert.NotContains(t, receivers, "filelog/varlog")

	journald := receivers["journald"].(map[string]any)
	assert.Equal(t, "warning", journald["priority"])
	assert.Equal(t, "end", journald["start_at"])
	assert.Equal(t, []any{
		map[string]any{"type": "filter", "expr": `body["_SYSTEMD_UNIT"] != nil and body["_SYSTEMD_UNIT"] matches "-collector-agent\\.service$"`},
		map[string]any{"type": "copy", "if": `body["_SYSTEMD_UNIT"] != nil and body["_SYSTEMD_UNIT"] != ""`, "from": "body._SYSTEMD_UNIT", "to": `resource["service.name"]`},
		map[string]any{"type": "copy", "if": `resource["service.name"] == nil and body["SYSLOG_IDENTIFIER"] != nil`, "from": "body.SYSLOG_IDENTIFIER", "to": `resource["service.name"]`},
		map[string]any{"type": "regex_replace", "field": `resource["service.name"]`, "regex": `\.service$`, "replace_with": ""},
	}, journald["operators"])

	logs := parsed["service"].(map[string]any)["pipelines"].(map[string]any)["logs"].(map[string]any)
	assert.Equal(t, []any{"journald"}, logs["receivers"])

	resourcedetection := parsed["processors"].(map[string]any)["resourcedetection"].(map[string]any)
	assert.Equal(t, []any{"env", "ec2", "gcp", "azure", "system"}, resourcedetection["detectors"])
	assert.Equal(t, false, resourcedetection["override"])
	assert.Equal(t, map[string]any{"host.id": map[string]any{"enabled": true}}, resourcedetection["system"].(map[string]any)["resource_attributes"])
}
