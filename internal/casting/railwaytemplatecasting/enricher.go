package railwaytemplatecasting

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*railwayTemplateMoldingEnricher)(nil)

type railwayTemplateMoldingEnricher struct {
	material []domain.StructuredMaterial
}

func newRailwayTemplateMoldingEnricher(config *v1alpha1.Casting) (*railwayTemplateMoldingEnricher, error) {
	material, err := getRailwayMaterial(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get compose yaml material: %w", err)
	}
	return &railwayTemplateMoldingEnricher{material: material}, nil
}

// railwayInternalHost returns the Railway private DNS hostname for a service.
// Railway services communicate via SERVICE_NAME.railway.internal within the same project.
func railwayInternalHost(serviceName string) string {
	return serviceName + ".railway.internal"
}

func (enricher *railwayTemplateMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	spec := config.SigNozSpec()

	name := config.Metadata.Name
	if name == "" {
		name = "signoz"
	}
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		if !spec.TelemetryStore.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-telemetrystore-" + spec.TelemetryStore.Kind.String()
		spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9000).String()}
		if spec.TelemetryStore.Status.Extras == nil {
			spec.TelemetryStore.Status.Extras = make(map[string]string)
		}
		spec.TelemetryStore.Status.Extras["service_names"] = svc
		spec.TelemetryStore.Status.Extras["_overrides"] = string(enricher.material[1].FmtContents())

	case v1alpha1.MoldingKindSignoz:
		if !spec.Signoz.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-signoz"
		spec.Signoz.Status.Addresses.APIServer = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 8080).String()}
		spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", railwayInternalHost(svc), 4320).String()}

	case v1alpha1.MoldingKindTelemetryKeeper:
		if !spec.TelemetryKeeper.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-telemetrykeeper-" + spec.TelemetryKeeper.Kind.String()
		spec.TelemetryKeeper.Status.Addresses.Client = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9181).String()}
		spec.TelemetryKeeper.Status.Addresses.Raft = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9234).String()}
		if spec.TelemetryKeeper.Status.Extras == nil {
			spec.TelemetryKeeper.Status.Extras = make(map[string]string)
		}
		spec.TelemetryKeeper.Status.Extras["service_names"] = svc
		spec.TelemetryKeeper.Status.Extras["_overrides"] = string(enricher.material[0].FmtContents())
	case v1alpha1.MoldingKindIngester:
		if !spec.Ingester.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-ingester"
		spec.Ingester.Status.Addresses.OTLP = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 4318).String()}
	}

	return nil
}
