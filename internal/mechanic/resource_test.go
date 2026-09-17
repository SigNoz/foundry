package mechanic

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestParseResource(t *testing.T) {
	tests := []struct {
		name                 string
		args                 []string
		pass                 bool
		expectedKind         v1alpha1.Kind
		expectedKindExplicit bool
		expectedMolding      v1alpha1.MoldingKind
		expectedEntityKind   string
		expectedEntityID     string
	}{
		{
			name:               "Positional_ImplicitKind",
			args:               []string{"signoz", "alert", "019c8af3"},
			pass:               true,
			expectedMolding:    v1alpha1.MoldingKindSignoz,
			expectedEntityKind: "alert",
			expectedEntityID:   "019c8af3",
		},
		{
			name:               "Slash_ImplicitKind",
			args:               []string{"signoz/alert/019c8af3"},
			pass:               true,
			expectedMolding:    v1alpha1.MoldingKindSignoz,
			expectedEntityKind: "alert",
			expectedEntityID:   "019c8af3",
		},
		{
			name:                 "Positional_ExplicitKind",
			args:                 []string{"installation", "signoz", "alert", "019c8af3"},
			pass:                 true,
			expectedKind:         v1alpha1.KindInstallation,
			expectedKindExplicit: true,
			expectedMolding:      v1alpha1.MoldingKindSignoz,
			expectedEntityKind:   "alert",
			expectedEntityID:     "019c8af3",
		},
		{
			name:                 "Slash_ExplicitKind",
			args:                 []string{"installation/signoz/alert/019c8af3"},
			pass:                 true,
			expectedKind:         v1alpha1.KindInstallation,
			expectedKindExplicit: true,
			expectedMolding:      v1alpha1.MoldingKindSignoz,
			expectedEntityKind:   "alert",
			expectedEntityID:     "019c8af3",
		},
		{
			name:                 "ExplicitKind_CaseInsensitive",
			args:                 []string{"Installation", "signoz", "alert", "019c8af3"},
			pass:                 true,
			expectedKind:         v1alpha1.KindInstallation,
			expectedKindExplicit: true,
			expectedMolding:      v1alpha1.MoldingKindSignoz,
			expectedEntityKind:   "alert",
			expectedEntityID:     "019c8af3",
		},
		{
			name:                 "CollectionAgent_Collector",
			args:                 []string{"collectionagent/collector/exporter/clickhouse"},
			pass:                 true,
			expectedKind:         v1alpha1.KindCollectionAgent,
			expectedKindExplicit: true,
			expectedMolding:      v1alpha1.MoldingKindCollector,
			expectedEntityKind:   "exporter",
			expectedEntityID:     "clickhouse",
		},
		{
			name:               "MixedSlashAndPositional",
			args:               []string{"signoz/alert", "019c8af3"},
			pass:               true,
			expectedMolding:    v1alpha1.MoldingKindSignoz,
			expectedEntityKind: "alert",
			expectedEntityID:   "019c8af3",
		},
		{
			name:               "MoldingAndEntityKind_NoID",
			args:               []string{"telemetrystore", "table"},
			pass:               true,
			expectedMolding:    v1alpha1.MoldingKindTelemetryStore,
			expectedEntityKind: "table",
		},
		{
			name:            "MoldingOnly",
			args:            []string{"signoz"},
			pass:            true,
			expectedMolding: v1alpha1.MoldingKindSignoz,
		},
		{
			name:                 "ExplicitKindAndMoldingOnly",
			args:                 []string{"installation", "telemetrystore"},
			pass:                 true,
			expectedKind:         v1alpha1.KindInstallation,
			expectedKindExplicit: true,
			expectedMolding:      v1alpha1.MoldingKindTelemetryStore,
		},
		{
			name: "Empty_Invalid",
			args: []string{},
			pass: false,
		},
		{
			name: "KindOnly_Invalid",
			args: []string{"installation"},
			pass: false,
		},
		{
			name: "UnknownMolding_Invalid",
			args: []string{"postgres", "table", "foo"},
			pass: false,
		},
		{
			name: "TooManySegments_Invalid",
			args: []string{"installation", "signoz", "alert", "019c8af3", "extra"},
			pass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseResource(tt.args)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedKind, res.Kind)
			assert.Equal(t, tt.expectedKindExplicit, res.KindExplicit)
			assert.Equal(t, tt.expectedMolding, res.Molding)
			assert.Equal(t, tt.expectedEntityKind, res.EntityKind)
			assert.Equal(t, tt.expectedEntityID, res.EntityID)
		})
	}
}

func TestResourceResolve(t *testing.T) {
	tests := []struct {
		name         string
		resource     Resource
		lockKind     v1alpha1.Kind
		pass         bool
		expectedKind v1alpha1.Kind
	}{
		{
			name:         "ImplicitKind_FilledFromLock",
			resource:     Resource{Molding: v1alpha1.MoldingKindSignoz, EntityKind: "alert", EntityID: "x"},
			lockKind:     v1alpha1.KindInstallation,
			pass:         true,
			expectedKind: v1alpha1.KindInstallation,
		},
		{
			name:         "ExplicitKind_Kept",
			resource:     Resource{Kind: v1alpha1.KindInstallation, KindExplicit: true, Molding: v1alpha1.MoldingKindSignoz},
			lockKind:     v1alpha1.KindInstallation,
			pass:         true,
			expectedKind: v1alpha1.KindInstallation,
		},
		{
			name:     "MoldingNotInKind_Invalid",
			resource: Resource{Molding: v1alpha1.MoldingKindCollector},
			lockKind: v1alpha1.KindInstallation,
			pass:     false,
		},
		{
			name:         "CollectionAgent_Collector_Valid",
			resource:     Resource{Molding: v1alpha1.MoldingKindCollector},
			lockKind:     v1alpha1.KindCollectionAgent,
			pass:         true,
			expectedKind: v1alpha1.KindCollectionAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := tt.resource.Resolve(tt.lockKind)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedKind, resolved.Kind)
		})
	}
}
