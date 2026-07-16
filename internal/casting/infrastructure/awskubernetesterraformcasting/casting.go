package awskubernetesterraformcasting

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
)

// Data carries the resolved values the templates render.
type Data struct {
	Name         string
	ResourceKind string
	Persistent   bool
	NodeGroups   []DataNodeGroup
}

type DataNodeGroup struct {
	Name   string
	Count  int
	VCPUs  int
	Memory int
	Disk   int
}

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
	data, err := newData(config)
	if err != nil {
		return nil, err
	}

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
		material, err := item.template.Render(data, filepath.Join(rootcasting.DeploymentDir, item.path))
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

// newData resolves the resource requirement document into the values the
// templates render.
func newData(config infrastructure.Casting) (*Data, error) {
	doc := config.Spec.Resource.Status.Config.Data[resourcemolding.ResourceConfigName]
	if doc == "" {
		return nil, foundryerrors.Newf(foundryerrors.TypeInternal, "resource config %q is missing from the resource status", resourcemolding.ResourceConfigName)
	}

	resourceConfig := &infrastructure.ResourceConfig{}
	if err := domain.UnmarshalYAML([]byte(doc), resourceConfig); err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to unmarshal resource config")
	}

	data := &Data{
		Name:         config.Metadata.Name,
		ResourceKind: config.Spec.Resource.Kind.String(),
		Persistent:   resourceConfig.Storage.Persistent != nil && *resourceConfig.Storage.Persistent,
	}

	for _, group := range resourceConfig.NodeGroups {
		if group.Count == nil || group.VCPUs == nil || group.Memory == nil || group.Disk == nil {
			return nil, foundryerrors.Newf(foundryerrors.TypeInternal, "node group %q in resource config is incomplete", group.Name)
		}

		data.NodeGroups = append(data.NodeGroups, DataNodeGroup{
			Name:   group.Name,
			Count:  *group.Count,
			VCPUs:  *group.VCPUs,
			Memory: *group.Memory,
			Disk:   *group.Disk,
		})
	}

	return data, nil
}
