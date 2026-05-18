package railwaytemplatecasting

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.MoldingEnricher = (*railwayTemplateMoldingEnricher)(nil)

type railwayTemplateMoldingEnricher struct {
	material []domain.StructuredMaterial
}

func newRailwayTemplateMoldingEnricher(config *installation.Casting) (*railwayTemplateMoldingEnricher, error) {
	material, err := getRailwayMaterial(config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to get compose yaml material")
	}
	return &railwayTemplateMoldingEnricher{material: material}, nil
}

// railwayInternalHost returns the Railway private DNS hostname for a service.
// Railway services communicate via SERVICE_NAME.railway.internal within the same project.
func railwayInternalHost(serviceName string) string {
	return serviceName + ".railway.internal"
}

func (enricher *railwayTemplateMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *installation.Casting) error {
	name := config.Metadata.Name
	if name == "" {
		name = "signoz"
	}
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		if !config.Spec.TelemetryStore.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-telemetrystore-" + config.Spec.TelemetryStore.Kind.String()
		config.Spec.TelemetryStore.Status.Addresses.TCP = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9000).String()}
		if config.Spec.TelemetryStore.Status.Extras == nil {
			config.Spec.TelemetryStore.Status.Extras = make(map[string]string)
		}
		config.Spec.TelemetryStore.Status.Extras["service_names"] = svc
		config.Spec.TelemetryStore.Status.Extras["_overrides"] = string(enricher.material[1].FmtContents())

	case v1alpha1.MoldingKindSignoz:
		if !config.Spec.Signoz.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-signoz"
		config.Spec.Signoz.Status.Addresses.APIServer = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 8080).String()}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{domain.MustNewAddress("ws", railwayInternalHost(svc), 4320).String()}

	case v1alpha1.MoldingKindTelemetryKeeper:
		if !config.Spec.TelemetryKeeper.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-telemetrykeeper-" + config.Spec.TelemetryKeeper.Kind.String()
		config.Spec.TelemetryKeeper.Status.Addresses.Client = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9181).String()}
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 9234).String()}
		if config.Spec.TelemetryKeeper.Status.Extras == nil {
			config.Spec.TelemetryKeeper.Status.Extras = make(map[string]string)
		}
		config.Spec.TelemetryKeeper.Status.Extras["service_names"] = svc
		config.Spec.TelemetryKeeper.Status.Extras["_overrides"] = string(enricher.material[0].FmtContents())
	case v1alpha1.MoldingKindIngester:
		if !config.Spec.Ingester.Spec.IsEnabled() {
			return nil
		}
		svc := name + "-ingester"
		config.Spec.Ingester.Status.Addresses.OTLP = []string{domain.MustNewAddress("tcp", railwayInternalHost(svc), 4318).String()}
	}

	return nil
}
