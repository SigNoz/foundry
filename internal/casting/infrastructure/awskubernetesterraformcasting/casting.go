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
	Name           string
	Storage        string
	MinSize        int
	MaxSize        int
	MachineType    string
	CPU            int
	Memory         int
	RootVolumeSize int
	DataVolumeSize int
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

	// The resource molding validates the resolved document, and moldings run
	// before any casting forges, so every field dereferenced here is present.
	// A missing one is a foundry bug, not bad input: recoverRunE turns the
	// panic into TypeFatal with the stack, which points at the line rather
	// than reporting a group as vaguely "incomplete".
	for _, group := range resourceConfig.NodeGroups {
		node := DataNodeGroup{
			Name:           group.Name,
			Storage:        group.Storage.String(),
			MinSize:        *group.MinSize,
			MaxSize:        *group.MaxSize,
			MachineType:    group.MachineType,
			RootVolumeSize: *group.RootVolume.Size,
		}

		if group.CPU != nil {
			node.CPU = *group.CPU
		}

		if group.Memory != nil {
			node.Memory = *group.Memory
		}

		if group.Storage.RequiresDataVolume() {
			data.Persistent = true
			node.DataVolumeSize = *group.DataVolume.Size
		}

		data.NodeGroups = append(data.NodeGroups, node)
	}

	return data, nil
}
