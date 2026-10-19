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
	foundryerrors "github.com/signoz/foundry/internal/errors"
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

	detectors, err := agent.GetStringSlice("processors.resourcedetection.detectors")
	require.NoError(t, err)

	assert.Equal(t, []string{"env", "ec2", "gcp", "azure", "system", "docker"}, detectors)

	devices, err := agent.GetStringSlice("receivers.hostmetrics.scrapers.disk.exclude.devices")
	require.NoError(t, err)

	assert.Equal(t, []string{`^ram\d+$`, `^zram\d+$`, `^loop\d+$`, `^fd\d+$`, `^hd[a-z]\d+$`, `^sd[a-z]\d+$`, `^vd[a-z]\d+$`, `^xvd[a-z]\d+$`, `^nvme\d+n\d+p\d+$`}, devices)

	mountPoints, err := agent.GetStringSlice("receivers.hostmetrics.scrapers.filesystem.exclude_mount_points.mount_points")
	require.NoError(t, err)

	assert.Equal(t, []string{"/dev/*", "/proc/*", "/sys/*", "/run/*", "/run/credentials/*", "/run/k3s/containerd/*", "/var/lib/docker/*", "/var/lib/containers/storage/*", "/var/lib/kubelet/*", "/snap/*"}, mountPoints)

	interfaces, err := agent.GetStringSlice("receivers.hostmetrics.scrapers.network.exclude.interfaces")
	require.NoError(t, err)

	assert.Equal(t, []string{"^veth.*$", "^docker.*$", "^br-.*$", "^flannel.*$", "^cali.*$", "^cbr.*$", "^cni.*$", "^dummy.*$", "^tailscale.*$", "^lo$"}, interfaces)

	regexes, err := agent.GetStringSlice(`receivers.filelog.operators.#(type=="regex_parser")#.regex`)
	require.NoError(t, err)

	assert.Equal(t, []string{`/var/lib/docker/containers/(?P<container_id>[a-f0-9]{64})/`}, regexes)

	moves, err := agent.GetStringSlice(`receivers.filelog.operators.#(type=="move")#.to`)
	require.NoError(t, err)

	assert.Equal(t, []string{`resource["container.id"]`}, moves)
}

func TestForge(t *testing.T) {
	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		pass              bool
		expectedErrorType int
	}{
		{"Agent_Valid", collectionagent.CollectorKindAgent, true, 0},
		{"Deployment_Invalid", collectionagent.CollectorKindDeployment, false, foundryerrors.TypeUnsupported.ExitCode()},
		{"Sidecar_Invalid", collectionagent.CollectorKindSidecar, false, foundryerrors.TypeUnsupported.ExitCode()},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := *collectionagent.Default()
			config.Spec.Collector.Kind = test.kind

			err := New(slog.New(slog.DiscardHandler)).Forge(t.Context(), config, pourer.New("collectionagent"))

			if test.pass {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))
		})
	}
}

// One agent per host reads that host's containers, so a replica count has
// nothing to scale.
func TestReplicas(t *testing.T) {
	for _, test := range []struct {
		name              string
		replicas          *int
		disabled          bool
		pass              bool
		expectedErrorType int
	}{
		{"Unstated_Valid", nil, false, true, 0},
		{"One_Valid", domain.NewIntPtr(1), false, true, 0},
		{"Two_Invalid", domain.NewIntPtr(2), false, false, foundryerrors.TypeUnsupported.ExitCode()},
		{"Disabled_Invalid", nil, true, false, foundryerrors.TypeUnsupported.ExitCode()},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := collectionagent.Default()
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas
			config.Spec.Collector.Spec.Enabled = v1alpha1.BoolPtr(!test.disabled)

			err := newDockerSwarmMoldingEnricher().EnrichStatus(t.Context(), v1alpha1.MoldingKindCollector, config)

			if test.pass {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))
		})
	}
}
