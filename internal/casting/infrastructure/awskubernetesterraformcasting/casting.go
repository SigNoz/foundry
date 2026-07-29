package awskubernetesterraformcasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	infrastructurecasting "github.com/signoz/foundry/internal/casting/infrastructure/casting"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
	"github.com/signoz/foundry/internal/pourer"
)

// Data carries the resolved values the templates render.
type Data struct {
	Name         string
	ResourceKind string
	Persistent   bool
	NodeGroups   []DataNodeGroup
}

// DataNodeGroup carries one node group's criteria for the templates.
type DataNodeGroup struct {
	Name   string
	Count  int
	VCPUs  int
	Memory int
	Disk   int
}

var _ infrastructurecasting.Casting = (*awsKubernetesTerraformCasting)(nil)

type awsKubernetesTerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *awsKubernetesTerraformCasting {
	return &awsKubernetesTerraformCasting{logger: logger}
}

func (c *awsKubernetesTerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return newAwsKubernetesTerraformMoldingEnricher(), nil
}

func (c *awsKubernetesTerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, p *pourer.Pourer) error {
	data, err := newData(config)
	if err != nil {
		return err
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

	for _, item := range items {
		buf := bytes.NewBuffer(nil)
		if err := item.template.Execute(buf, data); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute %s template", item.path)
		}

		p.AddJSON(buf.Bytes(), item.path)
	}

	return nil
}

func (c *awsKubernetesTerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.WarnContext(ctx, "casting the infrastructure is not implemented yet, run terraform init and apply from the pours directory", slog.String("path", filepath.Join(outputPath, p.Dir())))
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
	}

	for _, group := range resourceConfig.NodeGroups {
		if group.Count == nil || group.VCPUs == nil || group.Memory == nil || group.Disk == nil {
			return nil, foundryerrors.Newf(foundryerrors.TypeInternal, "node group %q in resource config is incomplete", group.Name)
		}

		if group.Persistent != nil && *group.Persistent {
			data.Persistent = true
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
