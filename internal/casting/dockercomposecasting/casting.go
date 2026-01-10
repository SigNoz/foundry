package dockercomposecasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/template"
	"github.com/signoz/foundry/internal/types"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
)

var _ casting.Casting = (*dockerComposeCasting)(nil)

type dockerComposeCasting struct {
	logger   *slog.Logger
	moldings map[v1alpha1.MoldingKind]*template.Template
	castings []*template.Template
}

func New(logger *slog.Logger) *dockerComposeCasting {
	return &dockerComposeCasting{
		logger: logger,
		castings: []*template.Template{
			composeYAMLTemplate,
		},
	}
}

func (casting *dockerComposeCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newDockerComposeMoldingEnricher(config)
}

func (casting *dockerComposeCasting) Forge(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	return nil, nil
}

func (casting *dockerComposeCasting) Cast(ctx context.Context, config v1alpha1.Casting) error {
	return nil
}

func getComposeMaterial(config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := composeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	return types.NewYAMLMaterial(buf.Bytes(), path)
}
