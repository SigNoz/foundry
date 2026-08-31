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

func TestParseOwner(t *testing.T) {
	tests := []struct {
		name          string
		cat           string
		expectedOwner domain.Owner
	}{
		{name: "NoSection_Zero", cat: "[Unit]\nDescription=x\n", expectedOwner: domain.Owner{}},
		{
			name:          "FoundrySection_Parsed",
			cat:           "[Unit]\nDescription=x\n[X-Foundry]\nOwner=kind=CollectionAgent,name=signoz\n",
			expectedOwner: domain.Owner{"kind": "CollectionAgent", "name": "signoz"},
		},
		{name: "FoundrySectionNoOwnerLine_Zero", cat: "[X-Foundry]\nComment=nothing here\n", expectedOwner: domain.Owner{}},
		{name: "Empty_Zero", cat: "", expectedOwner: domain.Owner{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedOwner, parseOwner(tt.cat))
		})
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		pass    bool
	}{
		{name: "Unit_Valid", options: Options{Unit: "otelcol-contrib.service"}, pass: true},
		{name: "NoUnit_Invalid", options: Options{}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.pass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestReleaseValidate(t *testing.T) {
	tests := []struct {
		name    string
		release Release
		pass    bool
	}{
		{
			name:    "NameOwnerUnits_Valid",
			release: Release{Release: domain.Release{Name: "signoz", Owner: domain.Owner{"a": "b"}}, Units: []string{"a.service"}},
			pass:    true,
		},
		{name: "NoUnits_Invalid", release: Release{Release: domain.Release{Name: "signoz", Owner: domain.Owner{"a": "b"}}}, pass: false},
		{name: "NoName_Invalid", release: Release{Release: domain.Release{Owner: domain.Owner{"a": "b"}}, Units: []string{"a.service"}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()
			if tt.pass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
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
	assert.NoError(t, r.Restart(context.Background(), release))
	assert.NoError(t, r.Down(context.Background(), release))
}

func TestOwnerGuardsTheUnit(t *testing.T) {
	requireSystemd(t)

	const name = "systemdtooler-owner-test"

	unit := filepath.Join(t.TempDir(), name+".service")
	contents := "[Unit]\nDescription=systemdtooler owner test\n[Service]\nType=oneshot\nExecStart=/bin/true\nRemainAfterExit=yes\n[X-Foundry]\nOwner=" + domain.Owner{"foundry.signoz.io/kind": "Installation", "foundry.signoz.io/name": name}.String() + "\n"
	assert.NoError(t, os.WriteFile(unit, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.Settings = tooler.NewSettings(io.Discard)

	installation := Release{
		Release: domain.Release{Name: name, Owner: domain.Owner{"foundry.signoz.io/kind": "Installation", "foundry.signoz.io/name": name}},
		Units:   []string{unit},
	}

	agent := Release{
		Release: domain.Release{Name: name, Owner: domain.Owner{"foundry.signoz.io/kind": "CollectionAgent", "foundry.signoz.io/name": name}},
		Units:   []string{unit},
	}

	assert.NoError(t, r.Up(context.Background(), installation))
	t.Cleanup(func() { _ = r.Down(context.Background(), installation) })

	assert.Error(t, r.Up(context.Background(), agent))
	assert.Error(t, r.Down(context.Background(), agent))
	assert.NoError(t, r.Down(context.Background(), installation))
}

func TestCat(t *testing.T) {
	requireSystemd(t)

	unit := filepath.Join(t.TempDir(), "systemdtooler-cat-test.service")
	contents := "[Unit]\nDescription=systemdtooler cat test\n[Service]\nType=oneshot\nExecStart=/bin/true\nRemainAfterExit=yes\n"
	assert.NoError(t, os.WriteFile(unit, []byte(contents), 0o644))

	r := New(slog.New(slog.DiscardHandler))
	r.Settings = tooler.NewSettings(io.Discard)

	release := Release{
		Release: domain.Release{Name: "systemdtooler-cat-test", Owner: domain.Owner{"foundry.signoz.io/managed-by": "foundry"}},
		Units:   []string{unit},
	}

	assert.NoError(t, r.Up(context.Background(), release))
	t.Cleanup(func() { _ = r.Down(context.Background(), release) })

	cat, err := r.Cat(context.Background(), Options{Unit: "systemdtooler-cat-test.service"})
	assert.NoError(t, err)
	assert.Contains(t, cat, "systemdtooler cat test")

	_, err = r.Cat(context.Background(), Options{Unit: "systemdtooler-does-not-exist.service"})
	assert.Error(t, err)
}
