package dockercomposecasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/internal/template"
	"github.com/signoz/foundry/internal/writer"

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
	}
}

func (casting *dockerComposeCasting) Forge(ctx context.Context, config v1alpha1.Casting) ([]writer.Material, error) {
	casting.logger.InfoContext(ctx, "forging docker compose files for stack", slog.String("casting.metadata.name", config.Metadata.Name))

	buf := bytes.NewBuffer(nil)
	err := ComposeYAMLTemplate.Execute(buf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to execute compose yaml template: %w", err)
	}

	fmt.Println(buf.String())

	return nil, nil
}

func (casting *dockerComposeCasting) Cast(ctx context.Context, config v1alpha1.Casting) error {
	casting.logger.InfoContext(ctx, "casting docker compose files for stack", slog.String("casting.metadata.name", config.Metadata.Name))

	return nil
}
