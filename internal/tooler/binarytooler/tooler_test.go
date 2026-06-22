package binarytooler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGauge_PassesWhenBinaryExists(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "signoz")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o755))

	require.NoError(t, New("signoz", path).Gauge(context.Background()))
}

func TestGauge_FailsWhenBinaryMissing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "does-not-exist")

	require.Error(t, New("signoz", path).Gauge(context.Background()))
}
