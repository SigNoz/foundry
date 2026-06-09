package mechanic

import (
	"context"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// execCall captures one invocation of the fake executor.
type execCall struct {
	name string
	args []string
}

// fakeExecutor records every command and returns canned output. Shared across
// the mechanic package tests (also used by telemetrystore_test.go).
type fakeExecutor struct {
	calls []execCall
	out   []byte
	err   error
}

func (f *fakeExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, execCall{name: name, args: args})
	return f.out, f.err
}

func dockerCasting(kind installation.MetaStoreKind) *installation.Casting {
	c := installation.Default()
	c.Spec.Deployment = v1alpha1.TypeDeployment{Mode: v1alpha1.ModeDocker, Flavor: v1alpha1.FlavorCompose}
	c.Spec.MetaStore.Kind = kind
	c.Spec.MetaStore.Status.Addresses.DSN = []string{"tcp://dev-metastore-postgres-0:5432"}
	c.Spec.Signoz.Status.Addresses.APIServer = []string{"tcp://dev-signoz-0:8080"}
	c.Spec.Signoz.Spec.Env = map[string]string{"SIGNOZ_SQLSTORE_SQLITE_PATH": "/var/lib/signoz/signoz.db"}
	return c
}

func TestNewMetaStoreUnsupported(t *testing.T) {
	c := installation.Default()
	c.Spec.Deployment = v1alpha1.TypeDeployment{Mode: v1alpha1.ModeKubernetes, Flavor: v1alpha1.FlavorHelm}

	_, err := NewMetaStore(&fakeExecutor{}, c)
	assert.Error(t, err)
}

func TestPostgresRuleQuery(t *testing.T) {
	exec := &fakeExecutor{out: []byte("019c8af3\x1f{\"alert\":\"High latency\"}\n")}

	store, err := NewMetaStore(exec, dockerCasting(installation.MetaStoreKindPostgres))
	require.NoError(t, err)

	alert, err := store.Rule(context.Background(), "019c8af3")
	require.NoError(t, err)

	assert.Equal(t, "019c8af3", alert.ID)
	assert.Equal(t, "High latency", alert.Name)
	assert.JSONEq(t, `{"alert":"High latency"}`, string(alert.Data))

	require.Len(t, exec.calls, 1)
	assert.Equal(t, "docker", exec.calls[0].name)
	assert.Equal(t, []string{
		"exec", "-e", "PGPASSWORD=signoz", "dev-metastore-postgres-0",
		"psql", "-U", "signoz", "-d", "signoz", "-tA", "-F", fieldSep,
		"-c", "SELECT id, data FROM rule WHERE id = '019c8af3'",
	}, exec.calls[0].args)
}

func TestSQLiteRuleQuery(t *testing.T) {
	exec := &fakeExecutor{out: []byte("42\x1f{\"alert\":\"Disk full\"}\n")}

	store, err := NewMetaStore(exec, dockerCasting(installation.MetaStoreKindSQLite))
	require.NoError(t, err)

	alert, err := store.Rule(context.Background(), "42")
	require.NoError(t, err)

	assert.Equal(t, "42", alert.ID)
	assert.Equal(t, "Disk full", alert.Name)

	require.Len(t, exec.calls, 2)
	assert.Equal(t, "docker", exec.calls[0].name)
	assert.Equal(t, []string{
		"exec", "-u", "root", "dev-signoz-0", "apk", "add", "--no-cache", "sqlite",
	}, exec.calls[0].args)
	assert.Equal(t, []string{
		"exec", "dev-signoz-0",
		"sqlite3", "-separator", fieldSep, "/var/lib/signoz/signoz.db",
		"SELECT id, data FROM rule WHERE id = '42'",
	}, exec.calls[1].args)
}

func TestRuleRejectsUnsafeID(t *testing.T) {
	store, err := NewMetaStore(&fakeExecutor{}, dockerCasting(installation.MetaStoreKindPostgres))
	require.NoError(t, err)

	_, err = store.Rule(context.Background(), "1' OR '1'='1")
	assert.Error(t, err)
}

func TestRuleNotFound(t *testing.T) {
	exec := &fakeExecutor{out: []byte("\n")}

	store, err := NewMetaStore(exec, dockerCasting(installation.MetaStoreKindPostgres))
	require.NoError(t, err)

	_, err = store.Rule(context.Background(), "missing")
	assert.Error(t, err)
}
