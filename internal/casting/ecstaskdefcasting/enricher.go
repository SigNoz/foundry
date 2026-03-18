package ecstaskdefcasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*ecsMoldingEnricher)(nil)

type ecsMoldingEnricher struct{}

func newEcsMoldingEnricher() *ecsMoldingEnricher {
	return &ecsMoldingEnricher{}
}

func (enricher *ecsMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", "telemetrystore", 9000)}

	case v1alpha1.MoldingKindTelemetryKeeper:
		config.Spec.TelemetryKeeper.Status.Addresses.Client = []string{types.FormatAddress("tcp", "telemetrykeeper", 9181)}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = []string{types.FormatAddress("tcp", "telemetrykeeper", 9234)}

	case v1alpha1.MoldingKindMetaStore:
		config.Spec.MetaStore.Status.Addresses.DSN = []string{types.FormatAddress("tcp", "metastore", 5432)}

	case v1alpha1.MoldingKindSignoz:
		config.Spec.Signoz.Status.Addresses.APIServer = []string{types.FormatAddress("tcp", "signoz", 8080)}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("ws", "signoz", 4320)}
	}

	return nil
}
