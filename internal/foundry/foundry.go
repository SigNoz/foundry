package foundry

import (
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/casting/dockercomposecasting"
	"github.com/signoz/foundry/internal/casting/linuxcasting"
	"github.com/signoz/foundry/internal/loader"
	"github.com/signoz/foundry/internal/loader/yamlloader"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/ingestermolding"
	"github.com/signoz/foundry/internal/molding/metastoremolding"
	"github.com/signoz/foundry/internal/molding/signozmolding"
	"github.com/signoz/foundry/internal/molding/telemetrykeepermolding"
	"github.com/signoz/foundry/internal/molding/telemetrystoremolding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
	"github.com/signoz/foundry/internal/tooler/dockertooler"
)

type Foundry struct {
	// Loader for loading the casting configuration.
	Loader loader.Loader

	// Logger for logging.
	Logger *slog.Logger

	// Castings for the different deployment modes.
	Castings map[string]casting.Casting

	// Toolers for the different deployment modes.
	Toolers map[string][]tooler.Tooler

	// Moldings for the different molding kinds.
	Moldings map[v1alpha1.MoldingKind]molding.Molding
}

func New(logger *slog.Logger) (*Foundry, error) {
	yamlLoader := yamlloader.New()

	return &Foundry{
		Loader: yamlLoader,
		Logger: logger,
		Castings: map[string]casting.Casting{
			"docker": dockercomposecasting.New(logger),
			"linux": linuxcasting.New(logger),
		},
		Toolers: map[string][]tooler.Tooler{
			"docker": {dockertooler.New(), dockercomposetooler.New()},
		},
		Moldings: map[v1alpha1.MoldingKind]molding.Molding{
			v1alpha1.MoldingKindTelemetryStore:  telemetrystoremolding.New(logger),
			v1alpha1.MoldingKindTelemetryKeeper: telemetrykeepermolding.New(logger),
			v1alpha1.MoldingKindMetaStore:       metastoremolding.New(logger),
			v1alpha1.MoldingKindSignoz:          signozmolding.New(logger),
			v1alpha1.MoldingKindIngester:        ingestermolding.New(logger),
		},
	}, nil
}

func (foundry *Foundry) CastingByDeploymentMode(deploymentMode string) (casting.Casting, error) {
	casting, ok := foundry.Castings[deploymentMode]
	if !ok {
		return nil, fmt.Errorf("deployment mode '%s' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this mode", deploymentMode)
	}

	return casting, nil
}

func (foundry *Foundry) ToolersByDeploymentMode(deploymentMode string) ([]tooler.Tooler, error) {
	toolers, ok := foundry.Toolers[deploymentMode]
	if !ok {
		return nil, fmt.Errorf("deployment mode '%s' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this mode", deploymentMode)
	}

	return toolers, nil
}
