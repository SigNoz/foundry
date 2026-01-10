package foundry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/casting/dockercomposecasting"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/loader"
	"github.com/signoz/foundry/internal/loader/yamlloader"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/signozmolding"
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
		},
		Toolers: map[string][]tooler.Tooler{
			"docker": {dockertooler.New(), dockercomposetooler.New()},
		},
		Moldings: map[v1alpha1.MoldingKind]molding.Molding{
			v1alpha1.MoldingKindTelemetryStore:  signozmolding.New(logger),
			v1alpha1.MoldingKindTelemetryKeeper: signozmolding.New(logger),
			v1alpha1.MoldingKindMetaStore:       signozmolding.New(logger),
			v1alpha1.MoldingKindSignoz:          signozmolding.New(logger),
			v1alpha1.MoldingKindIngester:        signozmolding.New(logger),
		},
	}, nil
}

func (foundry *Foundry) CastingByDeploymentMode(deploymentMode string) (casting.Casting, error) {
	casting, ok := foundry.Castings[deploymentMode]
	if !ok {
		return nil, fmt.Errorf("deployment mode '%s' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this mode.", deploymentMode)
	}

	return casting, nil
}

func (foundry *Foundry) ToolersByDeploymentMode(deploymentMode string) ([]tooler.Tooler, error) {
	toolers, ok := foundry.Toolers[deploymentMode]
	if !ok {
		return nil, fmt.Errorf("deployment mode '%s' is not supported, raise an issue at https://github.com/signoz/foundry/issues to request support for this mode.", deploymentMode)
	}

	return toolers, nil
}

func (foundry *Foundry) Gauge(ctx context.Context, config v1alpha1.Casting) error {
	toolers, err := foundry.ToolersByDeploymentMode(config.Spec.Deployment.Mode)
	if err != nil {
		return err
	}

	unavailableTools := []string{}

	for _, tooler := range toolers {
		err := tooler.Gauge(ctx)
		if err != nil {
			foundry.Logger.ErrorContext(ctx, "tool '%s' is not available or cannot be detected properly", tooler.Name(), foundryerrors.LogAttr(err))
			unavailableTools = append(unavailableTools, tooler.Name())
			continue
		}

		foundry.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", tooler.Name()))
	}

	if len(unavailableTools) > 0 {
		return fmt.Errorf("tools are not available, please install them and try again: %s", strings.Join(unavailableTools, ", "))
	}

	return nil
}
