package awskubernetesterraformcasting

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

type awsKubernetesTerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *awsKubernetesTerraformCasting {
	return &awsKubernetesTerraformCasting{logger: logger}
}

func (c *awsKubernetesTerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return &enricher{logger: c.logger}, nil
}

func (c *awsKubernetesTerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, poursPath string) ([]domain.Material, error) {
	items := []struct {
		template *domain.Template
		path     string
	}{
		{providersTFTemplate, "providers.tf.json"},
		{mainTFTemplate, "main.tf.json"},
		{variablesTFTemplate, "variables.tf.json"},
		{outputsTFTemplate, "outputs.tf.json"},
	}

	materials := make([]domain.Material, 0, len(items))
	for _, item := range items {
		material, err := item.template.Render(config, filepath.Join(rootcasting.DeploymentDir, item.path))
		if err != nil {
			return nil, err
		}
		materials = append(materials, material)
	}

	return materials, nil
}

func (c *awsKubernetesTerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, poursPath string) error {
	c.logger.WarnContext(ctx, "casting the infrastructure is not implemented yet, run terraform init and apply from the pours directory", slog.String("path", filepath.Join(poursPath, rootcasting.DeploymentDir)))
	return nil
}
