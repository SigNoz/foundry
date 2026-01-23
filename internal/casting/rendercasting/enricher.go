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

		var telemetrystoreAddresses []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "telemetrystore") {
				telemetrystoreAddresses = append(telemetrystoreAddresses, types.FormatAddress("tcp", serviceName, 9000))
			}
		}
		config.Spec.TelemetryStore.Status.Addresses.TCP = telemetrystoreAddresses

	case v1alpha1.MoldingKindSignoz:
		// For Render, we use service names for addresses
		// Service names follow format: name-signoz-N
		serviceName := fmt.Sprintf("%s-signoz", config.Metadata.Name)
		address := types.FormatAddress("http", serviceName, 8080)
		config.Spec.Signoz.Status.Addresses.APIServer = []string{address}
		config.Spec.Signoz.Status.Addresses.Opamp = []string{address}

	case v1alpha1.MoldingKindTelemetryKeeper:
		// Get telemetrykeeper service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get telemetrykeeper service names: %w", err)
		}

		var telemetrykeeperClientAddresses []string
		var telemetrykeeperRaftAddresses []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "telemetrykeeper") {
				telemetrykeeperClientAddresses = append(telemetrykeeperClientAddresses, types.FormatAddress("tcp", serviceName, 9181))
				telemetrykeeperRaftAddresses = append(telemetrykeeperRaftAddresses, types.FormatAddress("tcp", serviceName, 9234))
			}
		}
		config.Spec.TelemetryKeeper.Status.Addresses.Client = telemetrykeeperClientAddresses
		config.Spec.TelemetryKeeper.Status.Addresses.Raft = telemetrykeeperRaftAddresses

	case v1alpha1.MoldingKindIngester:
		// Get ingester service names
		serviceNames, err := enricher.material.GetStringSlice("services.#.name")
		if err != nil {
			return fmt.Errorf("failed to get ingester service names: %w", err)
		}

		var ingesterAddresses []string
		for _, serviceName := range serviceNames {
			if strings.Contains(serviceName, "ingester") {
				ingesterAddresses = append(ingesterAddresses, types.FormatAddress("tcp", serviceName, 4318))
			}
		}
		config.Spec.Ingester.Status.Addresses.OTLP = ingesterAddresses
	}

	return nil
}
