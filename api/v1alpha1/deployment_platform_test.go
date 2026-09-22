package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformDokployIsAvailable(t *testing.T) {
	var platform Platform
	require.NoError(t, platform.UnmarshalText([]byte("dokploy")))
	require.Equal(t, "dokploy", platform.String())
	require.Contains(t, Platforms(), PlatformDokploy)
}
