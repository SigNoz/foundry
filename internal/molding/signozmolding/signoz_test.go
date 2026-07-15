package signozmolding

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPostgresCasting(env map[string]string) *installation.Casting {
	config := &installation.Casting{}
	config.Spec.TelemetryStore.Status.Addresses.TCP = []string{"tcp://signoz-telemetrystore-clickhouse-0:9000"}
	config.Spec.MetaStore.Kind = installation.MetaStoreKindPostgres
	config.Spec.MetaStore.Status.Addresses.DSN = []string{"tcp://signoz-metastore-postgres-0:5432"}
	config.Spec.MetaStore.Status.Env = env
	return config
}

func TestPostgresDSN(t *testing.T) {
	molding := New(slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Run("default credentials render unchanged", func(t *testing.T) {
		config := newPostgresCasting(map[string]string{
			"POSTGRES_USER":     "signoz",
			"POSTGRES_PASSWORD": "signoz",
			"POSTGRES_DB":       "signoz",
		})
		require.NoError(t, molding.MoldV1Alpha1(context.Background(), config))
		assert.Equal(t,
			"postgres://signoz:signoz@signoz-metastore-postgres-0:5432/signoz?sslmode=disable",
			config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"],
		)
	})

	t.Run("credentials with reserved characters are percent-encoded", func(t *testing.T) {
		password := "p@ss/w0rd?x#%"
		config := newPostgresCasting(map[string]string{
			"POSTGRES_USER":     "signoz",
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       "signoz",
		})
		require.NoError(t, molding.MoldV1Alpha1(context.Background(), config))

		parsed, err := url.Parse(config.Spec.Signoz.Status.Env["SIGNOZ_SQLSTORE_POSTGRES_DSN"])
		require.NoError(t, err)
		parsedPassword, _ := parsed.User.Password()
		assert.Equal(t, password, parsedPassword)
		assert.Equal(t, "signoz", parsed.User.Username())
		assert.Equal(t, "signoz-metastore-postgres-0:5432", parsed.Host)
		assert.Equal(t, "/signoz", parsed.Path)
		assert.Equal(t, "sslmode=disable", parsed.RawQuery)
	})
}
