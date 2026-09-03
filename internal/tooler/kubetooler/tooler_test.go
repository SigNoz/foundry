package kubetooler

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/discovery"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
)

// requireCluster resolves the connection the tooler's own way, so a test never
// passes against a cluster the verbs could not have reached.
func requireCluster(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping kubernetes test in short mode")
	}

	config, err := New(slog.New(slog.DiscardHandler)).restConfig(tooler.Connection{})
	if err != nil {
		t.Skip("no kubeconfig resolved")
	}

	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		t.Skip("no kubernetes client")
	}

	if _, err := client.ServerVersion(); err != nil {
		t.Skip("cluster is not reachable")
	}
}

type otherTooler struct{}

func (otherTooler) Name() string                  { return "other" }
func (otherTooler) Gauge(_ context.Context) error { return nil }

func TestLookup(t *testing.T) {
	kube := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Only_Valid", toolers: []tooler.Tooler{kube}, pass: true},
		{name: "AmongOthers_Valid", toolers: []tooler.Tooler{otherTooler{}, kube}, pass: true},
		{name: "Empty_Invalid", toolers: nil, pass: false},
		{name: "OnlyOthers_Invalid", toolers: []tooler.Tooler{otherTooler{}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.toolers)

			if !tt.pass {
				assert.Error(t, err)
				assert.Nil(t, found)

				return
			}

			assert.NoError(t, err)
			assert.Same(t, kube, found)
		})
	}
}

func TestValidate(t *testing.T) {
	owner := domain.Owner{"foundry.signoz.io/name": "signoz"}

	complete := Release{
		Release:      domain.Release{Name: "signoz", Owner: owner},
		Namespace:    "signoz",
		Dir:          "pours/deployment",
		FieldManager: "foundry-installation",
	}

	without := func(mutate func(*Release)) Release {
		release := complete
		mutate(&release)

		return release
	}

	tests := []struct {
		name    string
		release Release
		pass    bool
	}{
		{name: "Complete_Valid", release: complete, pass: true},
		{name: "UnstatedName_Invalid", release: without(func(r *Release) { r.Name = "" })},
		{name: "UnstatedOwner_Invalid", release: without(func(r *Release) { r.Owner = nil })},
		{name: "UnstatedNamespace_Invalid", release: without(func(r *Release) { r.Namespace = "" })},
		{name: "UnstatedDirectory_Invalid", release: without(func(r *Release) { r.Dir = "" })},
		{name: "UnstatedFieldManager_Invalid", release: without(func(r *Release) { r.FieldManager = "" })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()

			if !tt.pass {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestUndeletable(t *testing.T) {
	tests := []struct {
		name             string
		kind             string
		expectedStanding bool
	}{
		{name: "Namespace_Standing", kind: "Namespace", expectedStanding: true},
		{name: "PersistentVolumeClaim_Standing", kind: "PersistentVolumeClaim", expectedStanding: true},
		{name: "CustomResourceDefinition_Standing", kind: "CustomResourceDefinition", expectedStanding: true},
		{name: "ConfigMap_Removed", kind: "ConfigMap"},
		{name: "Empty_Removed", kind: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedStanding, undeletable(tt.kind))
		})
	}
}

func root(t *testing.T, name string, owner domain.Owner) string {
	t.Helper()

	dir := t.TempDir()

	pairs := strings.Builder{}
	for _, key := range []string{"foundry.signoz.io/kind", "foundry.signoz.io/managed-by", "foundry.signoz.io/name"} {
		if value, ok := owner[key]; ok {
			pairs.WriteString("    " + key + ": " + value + "\n")
		}
	}

	kustomization := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n" +
		"resources:\n- configmap.yaml\n" +
		"labels:\n- includeSelectors: false\n  pairs:\n" + pairs.String()

	configmap := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + name + "\ndata:\n  ok: \"true\"\n"

	require.NoError(t, os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(kustomization), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "configmap.yaml"), []byte(configmap), 0o644))

	return dir
}

func TestRender(t *testing.T) {
	owner := domain.Owner{"foundry.signoz.io/name": "kubetooler-test"}

	tests := []struct {
		name         string
		dir          string
		pass         bool
		expectedKind string
		expectedName string
	}{
		{
			name:         "Root_Valid",
			dir:          root(t, "kubetooler-test", owner),
			pass:         true,
			expectedKind: "ConfigMap",
			expectedName: "kubetooler-test",
		},
		{
			name: "UnstatedRoot_Invalid",
			dir:  filepath.Join(t.TempDir(), "absent"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects, err := render(tt.dir)
			if !tt.pass {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Len(t, objects, 1)

			assert.Equal(t, tt.expectedKind, objects[0].GetKind())
			assert.Equal(t, tt.expectedName, objects[0].GetName())
			assert.Equal(t, tt.expectedName, objects[0].GetLabels()["foundry.signoz.io/name"])
		})
	}
}

func TestApplyDelete(t *testing.T) {
	requireCluster(t)

	const name = "kubetooler-test"

	owner := domain.Owner{
		"foundry.signoz.io/managed-by": "foundry",
		"foundry.signoz.io/kind":       "Installation",
		"foundry.signoz.io/name":       name,
	}

	kube := New(slog.New(slog.DiscardHandler))

	release := Release{
		Release:      domain.Release{Name: name, Owner: owner},
		Namespace:    "default",
		Dir:          root(t, name, owner),
		FieldManager: "foundry-installation",
	}

	require.NoError(t, kube.Apply(context.Background(), release))
	t.Cleanup(func() { _ = kube.Delete(context.Background(), release) })

	ownership, err := kube.Owners(context.Background(), release)
	require.NoError(t, err)

	_, conflict := ownership.Foreign(owner)
	assert.False(t, conflict)

	assert.NoError(t, kube.Delete(context.Background(), release))
}

// An object labelled for one owner is refused to another, and granted back to
// the owner that holds it.
func TestOwnerGuardsTheRelease(t *testing.T) {
	requireCluster(t)

	const name = "kubetooler-owner-test"

	owner := domain.Owner{
		"foundry.signoz.io/managed-by": "foundry",
		"foundry.signoz.io/kind":       "Installation",
		"foundry.signoz.io/name":       name,
	}

	kube := New(slog.New(slog.DiscardHandler))

	installation := Release{
		Release:      domain.Release{Name: name, Owner: owner},
		Namespace:    "default",
		Dir:          root(t, name, owner),
		FieldManager: "foundry-installation",
	}

	// One key of the set differing is a different owner.
	foreign := maps.Clone(owner)
	foreign["foundry.signoz.io/kind"] = "CollectionAgent"

	agent := installation
	agent.Owner = foreign
	agent.FieldManager = "foundry-collectionagent"

	require.NoError(t, kube.Apply(context.Background(), installation))
	t.Cleanup(func() { _ = kube.Delete(context.Background(), installation) })

	assert.ErrorContains(t, kube.Apply(context.Background(), agent), "already belongs to")
	assert.ErrorContains(t, kube.Delete(context.Background(), agent), "already belongs to")
	assert.NoError(t, kube.Delete(context.Background(), installation))
}
