package ecsec2terraformcasting

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	ecscontract "github.com/signoz/foundry/internal/contract/aws/ecs"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/terraformtooler"
)

type cloudInitData struct {
	Cluster    string
	Selector   map[string]string
	DataVolume bool
}

type ecsEc2TerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ecsEc2TerraformCasting {
	return &ecsEc2TerraformCasting{logger: logger}
}

func (c *ecsEc2TerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return newEcsEc2TerraformMoldingEnricher(), nil
}

func (c *ecsEc2TerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, p *pourer.Pourer) error {
	data, err := newResources(config)
	if err != nil {
		return err
	}

	items := []struct {
		template *domain.Template
		path     string
	}{
		{versionsTFTemplate, "versions.tf.json"},
		{backendTFTemplate, "backend.tf.json"},
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

	// Blobs stay byte-exact, preserving the #cloud-config header.
	boots := map[string]cloudInitData{}

	for key, group := range data.Pinned {
		boots[key] = cloudInitData{
			Cluster:    data.Cluster.Name,
			Selector:   group.Selector,
			DataVolume: group.Storage.RequiresDataVolume(),
		}
	}

	for key, group := range data.Pools {
		boots[key] = cloudInitData{
			Cluster:    data.Cluster.Name,
			Selector:   group.Selector,
			DataVolume: group.Storage.RequiresDataVolume(),
		}
	}

	for key, boot := range boots {
		buf := bytes.NewBuffer(nil)
		if err := cloudInitTemplate.Execute(buf, boot); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute cloud-init template")
		}

		p.AddBlob(buf.Bytes(), "cloud-init", key+".yaml")
	}

	return nil
}

func (c *ecsEc2TerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Apply(ctx, terraformtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Root:    filepath.Join(outputPath, p.Dir()),
	})
}

// Melt destroys the substrate, and the volumes it holds go with it.
func (c *ecsEc2TerraformCasting) Melt(ctx context.Context, config infrastructure.Casting, outputPath string, p *pourer.Pourer, toolers []tooler.Tooler) error {
	terraform, err := terraformtooler.Lookup(toolers)
	if err != nil {
		return err
	}

	return terraform.Destroy(ctx, terraformtooler.Release{
		Release: domain.Release{Name: config.Metadata.Name, Owner: config.Labels()},
		Root:    filepath.Join(outputPath, p.Dir()),
	})
}

func newResources(config infrastructure.Casting) (*ecscontract.Resources, error) {
	doc := config.Spec.Resource.Status.Config.Data[resourcemolding.ResourceConfigName]

	if doc == "" {
		return nil, foundryerrors.Newf(foundryerrors.TypeInternal, "resource config %q is missing from the resource status", resourcemolding.ResourceConfigName)
	}

	declaration := &infrastructure.ResourceConfig{}
	if err := domain.UnmarshalYAML([]byte(doc), declaration); err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to unmarshal resource config")
	}

	substrate, err := contract.NewSubstrate(config.Metadata.Name)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to resolve the substrate being provisioned")
	}

	return ecscontract.Derive(substrate, declaration, config.Labels())
}
