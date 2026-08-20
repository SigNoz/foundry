package terraformrunner

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/internal/runner"
	"github.com/stretchr/testify/assert"
)

// requireTerraform skips a test that drives the real terraform binary.
func requireTerraform(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping terraform test in short mode")
	}

	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform is not available")
	}
}

// root writes a provider-less terraform root, so apply and destroy touch
// nothing outside the directory.
func root(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	contents := `{"output": {"ok": {"value": "ok"}}}`
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf.json"), []byte(contents), 0o644))

	return dir
}

type otherRunner struct{}

func (otherRunner) Name() string                    { return "other" }
func (otherRunner) Gauge(ctx context.Context) error { return nil }

func TestNew(t *testing.T) {
	assert.Equal(t, "terraform", New(slog.New(slog.DiscardHandler)).Name())
}

func TestLookup(t *testing.T) {
	terraform := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		runners []runner.Runner
		pass    bool
	}{
		{name: "Registered_Found", runners: []runner.Runner{terraform}, pass: true},
		{name: "AmongOthers_Found", runners: []runner.Runner{otherRunner{}, terraform}, pass: true},
		{name: "Empty_Invalid", runners: nil, pass: false},
		{name: "OnlyOthers_Invalid", runners: []runner.Runner{otherRunner{}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.runners)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, terraform, found)
		})
	}
}

// The verbs validate what they execute against, so a casting that forgot to
// state the root, or forged nothing into it, fails before terraform is spawned.
func TestRunValidatesRoot(t *testing.T) {
	tests := []struct {
		name string
		root string
		pass bool
	}{
		{name: "Unset_Invalid", root: "", pass: false},
		{name: "Unforged_Invalid", root: t.TempDir(), pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(slog.New(slog.DiscardHandler))

			if !tt.pass {
				assert.Error(t, r.Apply(context.Background(), Options{Root: tt.root}))
				assert.Error(t, r.Destroy(context.Background(), Options{Root: tt.root}))
				return
			}

			assert.NoError(t, r.Apply(context.Background(), Options{Root: tt.root}))
		})
	}
}

// Apply then Destroy against a provider-less root: no cloud, no credentials.
func TestApplyDestroy(t *testing.T) {
	requireTerraform(t)

	dir := root(t)
	r := New(slog.New(slog.DiscardHandler))
	options := Options{Root: dir, Stdout: io.Discard, Stderr: io.Discard}

	assert.NoError(t, r.Gauge(context.Background()))
	assert.NoError(t, r.Apply(context.Background(), options))

	_, err := os.Stat(filepath.Join(dir, "terraform.tfstate"))
	assert.NoError(t, err)

	assert.NoError(t, r.Destroy(context.Background(), options))
}

// Apply runs init itself, so a root that was never initialised still applies.
func TestApplyInitialisesTheRoot(t *testing.T) {
	requireTerraform(t)

	dir := root(t)

	_, err := os.Stat(filepath.Join(dir, ".terraform"))
	assert.True(t, os.IsNotExist(err))

	r := New(slog.New(slog.DiscardHandler))
	assert.NoError(t, r.Apply(context.Background(), Options{Root: dir, Stdout: io.Discard, Stderr: io.Discard}))
}
