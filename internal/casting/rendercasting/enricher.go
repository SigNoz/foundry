package rendercasting

import (
	"context"
	"fmt"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

var _ molding.MoldingEnricher = (*renderMoldingEnricher)(nil)

type renderMoldingEnricher struct {
	material types.Material
}

func newRenderMoldingEnricher(config *v1alpha1.Casting) (*renderMoldingEnricher, error) {
	material, err := getRenderMaterial(config, "render.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to get render yaml material: %w", err)
	}

	return &renderMoldingEnricher{material: material}, nil
}

func (enricher *renderMoldingEnricher) EnrichStatus(ctx context.Context, kind v1alpha1.MoldingKind, config *v1alpha1.Casting) error {
	switch kind {
	case v1alpha1.MoldingKindTelemetryStore:
		// Get telemetrystore service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrystore service names: %w", err)
		}

		var addrs []string
		var storeServiceNames []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "telemetrystore") && !strings.Contains(serviceName, "migrator") {
				addrs = append(addrs, types.FormatAddress("tcp", serviceName, 9000))
				storeServiceNames = append(storeServiceNames, serviceName)
			}
		}
		config.Spec.TelemetryStore.Status.Addresses.TCP = addrs

		// Store service names in extras for template usage
		if config.Spec.TelemetryStore.Status.Extras == nil {
			config.Spec.TelemetryStore.Status.Extras = make(map[string]string)
		}
		config.Spec.TelemetryStore.Status.Extras["service_names"] = strings.Join(storeServiceNames, ",")

	case v1alpha1.MoldingKindSignoz:
		// Get telemetrystore service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrystore service names: %w", err)
		}

		var apiServerAddr []string
		var opampAddr []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "-signoz") {
				apiServerAddr = append(apiServerAddr, types.FormatAddress("tcp", serviceName, 8080))
				opampAddr = append(opampAddr, types.FormatAddress("ws", serviceName, 4320))
			}
		}
		config.Spec.Signoz.Status.Addresses.APIServer = apiServerAddr
		config.Spec.Signoz.Status.Addresses.Opamp = opampAddr

	case v1alpha1.MoldingKindTelemetryKeeper:
		// Get telemetrykeeper service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper service names: %w", err)
		}

		var addrsClient []string
		var addrsRaft []string
		var keeperServiceNames []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "telemetrykeeper") {
				addrsClient = append(addrsClient, types.FormatAddress("tcp", serviceName, 9181))
				addrsRaft = append(addrsRaft, types.FormatAddress("tcp", serviceName, 9234))
				keeperServiceNames = append(keeperServiceNames, serviceName)
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Client = addrsClient
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = addrsRaft

		// Store service names in extras for template usage
		if config.Spec.TelemetryKeeper.Status.Extras == nil {
			config.Spec.TelemetryKeeper.Status.Extras = make(map[string]string)
		}
		config.Spec.TelemetryKeeper.Status.Extras["service_names"] = strings.Join(keeperServiceNames, ",")

	case v1alpha1.MoldingKindIngester:
		// Get ingester service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get ingester service names: %w", err)
		}

		var addrs []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "ingester") {
				addrs = append(addrs, types.FormatAddress("tcp", serviceName, 4318))
			}
		}
		config.Spec.Ingester.Status.Addresses.OTLP = addrs
	}

	return nil
}
