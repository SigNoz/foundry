package dockerswarmcasting

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
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
	require.Contains(t, string(compose), "traefik.docker.network=dokploy-network")
	require.Contains(t, string(compose), "traefik.http.routers.signoz-signoz.rule=Host(`${SIGNOZ_DOMAIN:?SIGNOZ_DOMAIN_required}`)")
	require.Contains(t, string(compose), "traefik.http.services.signoz-signoz.loadbalancer.server.port=8080")
	require.Equal(t, 1, strings.Count(string(compose), "- dokploy-network"))
	require.Contains(t, paths, filepath.Join(rootcasting.DeploymentDir, "ingester", "ingester.yaml"))
	require.Contains(t, paths, filepath.Join(rootcasting.DeploymentDir, "ingester", "opamp.yaml"))
}

func TestDokployCastIsNoOp(t *testing.T) {
	config := installation.Default(&installation.Casting{})
	config.Spec.Deployment.Platform = v1alpha1.PlatformDokploy

	require.NoError(t, NewDokploy(slog.Default()).Cast(context.Background(), *config, t.TempDir()))
}
