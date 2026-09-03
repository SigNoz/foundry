package systemdtooler

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/stretchr/testify/assert"
)

func requireSystemd(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping systemd test in short mode")
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("systemctl is not available")
	}

	if os.Geteuid() != 0 {
		t.Skip("systemd test needs root")
	}
}

func TestNames(t *testing.T) {
	tests := []struct {
		name          string
		units         []string
		expectedNames []string
	}{
		{name: "None_Empty", units: nil, expectedNames: []string{}},
		{name: "PathsAndNames_Based", units: []string{"/etc/systemd/system/a.service", "b.service"}, expectedNames: []string{"a.service", "b.service"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedNames, names(tt.units))
		})
	}
}

func TestUpDown(t *testing.T) {
	requireSystemd(t)

	unit := filepath.Join(t.TempDir(), "systemdtooler-test.service")
	contents := "[Unit]\nDescription=systemdtooler test\n[Service]\nType=oneshot\nExecStart=/bin/true\nRemainAfterExit=yes\n"
	assert.NoError(t, os.WriteFile(unit, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.Settings = tooler.NewSettings(io.Discard)

	release := Release{
		Release: domain.Release{
			Name:  "systemdtooler-test",
			Owner: domain.Owner{"foundry.signoz.io/managed-by": "foundry"},
		},
		Units: []string{unit},
	}

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Up(context.Background(), release))
	t.Cleanup(func() { _ = r.Down(context.Background(), release) })
	assert.NoError(t, r.Down(context.Background(), release))
}
