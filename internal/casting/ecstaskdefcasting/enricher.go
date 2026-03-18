package ecstaskdefcasting

import (
	"context"
	"fmt"

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
		config.Spec.TelemetryStore.Status.Addresses.TCP = []string{types.FormatAddress("tcp", fmt.Sprintf("telemetrystore-%s", config.Spec.TelemetryStore.Kind.String()), 9000)}

	case v1alpha1.MoldingKindTelemetryKeeper:
		config.Spec.TelemetryKeeper.Status.Addresses.Client = []string{types.FormatAddress("tcp", fmt.Sprintf("telemetrykeeper-%s", config.Spec.TelemetryKeeper.Kind.String()), 9181)}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = []string{types.FormatAddress("tcp", fmt.Sprintf("telemetrykeeper-%s", config.Spec.TelemetryKeeper.Kind.String()), 9234)}

	case v1alpha1.MoldingKindMetaStore:
		config.Spec.MetaStore.Status.Addresses.DSN = []string{types.FormatAddress("tcp", fmt.Sprintf("metastore-%s", config.Spec.MetaStore.Kind.String()), 5432)}

	case v1alpha1.MoldingKindSignoz:
		config.Spec.Signoz.Status.Addresses.APIServer = []string{types.FormatAddress("tcp", "signoz", 8080)}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{types.FormatAddress("ws", "signoz", 4320)}
	}

	return nil
}
