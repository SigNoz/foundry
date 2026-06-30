// Package installation holds the compatibility matrix for the installation Kind
// and evaluates it through the generic compat engine.
package installation

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/compat"
)

// compatRules is the compatibility matrix for the installation Kind: one row
// per known breaking change between components.
var compatRules = []compat.Rule{
	{
		Subject:  v1alpha1.MoldingKindIngester,
		When:     ">0.144.5",
		Target:   v1alpha1.MoldingKindTelemetryStore,
		Requires: "=25.12.5",
		Advice:   "pin ingester to 0.144.5, or upgrade telemetrystore clickhouse to 25.12.5",
	},
}

// Compatibility resolves every molding's version from its image and
// validates the combination against the compatibility matrix.
func Compatibility(c *installation.Casting, logger *slog.Logger) error {
	versions := map[v1alpha1.MoldingKind]compat.Resolved{
		v1alpha1.MoldingKindTelemetryStore:  resolve(c.Spec.TelemetryStore.Spec),
		v1alpha1.MoldingKindTelemetryKeeper: resolve(c.Spec.TelemetryKeeper.Spec),
		v1alpha1.MoldingKindMetaStore:       resolve(c.Spec.MetaStore.Spec),
		v1alpha1.MoldingKindSignoz:          resolve(c.Spec.Signoz.Spec),
		v1alpha1.MoldingKindIngester:        resolve(c.Spec.Ingester.Spec),
		v1alpha1.MoldingKindMCP:             resolve(c.Spec.MCP.Spec),
	}

	return compat.Check(versions, compatRules, logger)
}

func resolve(spec v1alpha1.MoldingSpec) compat.Resolved {
	return compat.NewResolved(spec.Image, spec.IsEnabled())
}
