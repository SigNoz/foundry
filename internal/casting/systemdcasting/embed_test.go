package systemdcasting

import (
	"bytes"
	"log/slog"
	"path"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotEmptyAndValid(t *testing.T) {
	serviceTemplates := map[string]*domain.Template{
		"telemetryStoreServiceTemplate":  telemetryStoreServiceTemplate,
		"telemetryKeeperServiceTemplate": telemetryKeeperServiceTemplate,
		"zookeeperServiceTemplate":       zookeeperServiceTemplate,
		"zookeeperConfigTemplate":        zookeeperConfigTemplate,
		"metaStoreServiceTemplate":       metaStoreServiceTemplate,
		"signozServiceTemplate":          signozServiceTemplate,
		"ingesterServiceTemplate":        ingesterServiceTemplate,
		"telemetryStoreMigratorTemplate": telemetryStoreMigratorServiceTemplate,
	}

	for name, st := range serviceTemplates {
		assert.NotEmpty(t, st, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := st.Execute(buf, nil)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}

func TestTarget(t *testing.T) {
	for _, test := range []struct {
		name             string
		keeper           installation.TelemetryKeeperKind
		pass             bool
		expectedTarget   string
		expectedServices int
	}{
		{"Default_Valid", installation.TelemetryKeeperKindClickhouseKeeper, true, "signoz-installation.target", 7},
		{"Zookeeper_Valid", installation.TelemetryKeeperKindZookeeper, true, "signoz-installation.target", 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			declared := &installation.Casting{}
			declared.Spec.TelemetryKeeper.Kind = test.keeper

			cfg := installation.Default(declared)
			cfg.Spec.MCP.Spec.Enabled = v1alpha1.BoolPtr(true)

			materials, err := New(slog.New(slog.DiscardHandler)).Forge(t.Context(), *cfg, "")

			if test.pass {
				require.NoError(t, err)

				var targets []string
				services := 0

				for _, material := range materials {
					unit := path.Base(material.Path())

					if strings.HasSuffix(unit, targetSuffix) {
						targets = append(targets, unit)
					}

					if !strings.HasSuffix(unit, svcSuffix) {
						continue
					}

					services++

					contents := string(material.FmtContents())
					assert.Contains(t, contents, "PartOf="+test.expectedTarget, unit)
					assert.Contains(t, contents, "WantedBy="+test.expectedTarget, unit)
					assert.NotContains(t, contents, "WantedBy=multi-user.target", unit)
				}

				assert.Equal(t, []string{test.expectedTarget}, targets)
				assert.Equal(t, test.expectedServices, services)

				return
			}

			require.Error(t, err)
		})
	}
}
