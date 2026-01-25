package foundry

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/config/yamlconfig"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/ingestermolding"
	"github.com/signoz/foundry/internal/molding/metastoremolding"
	"github.com/signoz/foundry/internal/molding/signozmolding"
	"github.com/signoz/foundry/internal/molding/telemetrykeepermolding"
	"github.com/signoz/foundry/internal/molding/telemetrystoremolding"
)

type Foundry struct {
	// Config for loading the casting configuration.
	Config config.Config

	// Logger for logging.
	Logger *slog.Logger

	// Registry for the different deployments.
	Registry *Registry

	// Moldings for the different molding kinds.
	Moldings map[v1alpha1.MoldingKind]molding.Molding
}

func New(logger *slog.Logger) (*Foundry, error) {
	yamlConfig := yamlconfig.New()

	registry, err := NewRegistry(logger)
	if err != nil {
		return nil, err
	}

	return &Foundry{
		Config:   yamlConfig,
		Logger:   logger,
		Registry: registry,
		Moldings: map[v1alpha1.MoldingKind]molding.Molding{
			v1alpha1.MoldingKindTelemetryStore:  telemetrystoremolding.New(logger),
			v1alpha1.MoldingKindTelemetryKeeper: telemetrykeepermolding.New(logger),
			v1alpha1.MoldingKindMetaStore:       metastoremolding.New(logger),
			v1alpha1.MoldingKindSignoz:          signozmolding.New(logger),
			v1alpha1.MoldingKindIngester:        ingestermolding.New(logger),
		},
	}, nil
}
