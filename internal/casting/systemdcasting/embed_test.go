package systemdcasting

import (
	"bytes"
	"context"
	"log/slog"
	"path"
	"slices"
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

func TestUnitPatterns(t *testing.T) {
	tests := []struct {
		name          string
		keeper        installation.TelemetryKeeperKind
		forgedName    string
		downName      string
		expectedUnits int
		pass          bool
	}{
		{name: "ClickhouseKeeper_Valid", keeper: installation.TelemetryKeeperKindClickhouseKeeper, forgedName: "signoz", downName: "signoz", expectedUnits: 7, pass: true},
		{name: "Zookeeper_Valid", keeper: installation.TelemetryKeeperKindZookeeper, forgedName: "signoz", downName: "signoz", expectedUnits: 7, pass: true},
		{name: "LongerName_Invalid", keeper: installation.TelemetryKeeperKindClickhouseKeeper, forgedName: "signoz-other", downName: "signoz", expectedUnits: 7, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			declared := &installation.Casting{}
			declared.Spec.TelemetryKeeper.Kind = tt.keeper

			cfg := installation.Default(declared)
			cfg.Metadata.Name = tt.forgedName
			cfg.Spec.MCP.Spec.Enabled = v1alpha1.BoolPtr(true)

			materials, err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), *cfg, "")
			require.NoError(t, err)

			units := 0
			for _, material := range materials {
				unit := path.Base(material.Path())
				if !strings.HasSuffix(unit, svcSuffix) {
					continue
				}

				units++

				matched := slices.ContainsFunc(unitPatterns, func(pattern string) bool {
					ok, err := path.Match(tt.downName+"-"+pattern, unit)
					require.NoError(t, err)

					return ok
				})

				assert.Equal(t, tt.pass, matched, unit)
			}

			assert.Equal(t, tt.expectedUnits, units)
		})
	}
}
