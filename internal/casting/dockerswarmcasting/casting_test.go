package dockerswarmcasting

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/stretchr/testify/require"
)

func TestDokployForgeProducesValidStackMaterials(t *testing.T) {
	config := installation.Default(&installation.Casting{})
	config.Spec.Deployment.Platform = v1alpha1.PlatformDokploy
	config.Spec.Deployment.Flavor = v1alpha1.FlavorStack
	config.Spec.Ingester.Spec.Config.Data = map[string]string{
		"ingester.yaml": "service: {}",
		"opamp.yaml":    "server: {}",
	}

	materials, err := NewDokploy(slog.Default()).Forge(context.Background(), *config, t.TempDir())
	require.NoError(t, err)
	require.NotEmpty(t, materials)

	var compose []byte
	paths := make(map[string]struct{}, len(materials))
	for _, material := range materials {
		paths[material.Path()] = struct{}{}
		if material.Path() == filepath.Join(rootcasting.DeploymentDir, "compose.yaml") {
			compose = material.FmtContents()
		}
	}

	require.NotEmpty(t, compose)
	require.NotContains(t, string(compose), "container_name")
	require.NotContains(t, string(compose), "content:")
	require.NotContains(t, string(compose), "exclude_from_hc")
	require.Contains(t, string(compose), "name: dokploy-network")
	require.Contains(t, string(compose), "external: true")
	require.Contains(t, string(compose), "traefik.enable=true")
	require.Contains(t, paths, filepath.Join(rootcasting.DeploymentDir, "ingester", "ingester.yaml"))
	require.Contains(t, paths, filepath.Join(rootcasting.DeploymentDir, "ingester", "opamp.yaml"))
}
