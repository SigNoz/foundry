package awsecsterraformcasting

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	ecscontract "github.com/signoz/foundry/internal/contract/aws/ecs"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
	"github.com/signoz/foundry/internal/pourer"
)

type awsEcsTerraformCasting struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *awsEcsTerraformCasting {
	return &awsEcsTerraformCasting{logger: logger}
}

func (c *awsEcsTerraformCasting) Enricher(ctx context.Context, config *infrastructure.Casting) (infrastructuremolding.MoldingEnricher, error) {
	return newAwsEcsTerraformMoldingEnricher(), nil
}

func (c *awsEcsTerraformCasting) Forge(ctx context.Context, config infrastructure.Casting, p *pourer.Pourer) error {
	resources, err := c.resources(config)
	if err != nil {
		return err
	}

	data := struct {
		*ecscontract.Resources

		Region string
	}{resources, infrastructure.ECSRegion.Resolve(config.Metadata.Annotations)}

	for _, tmpl := range []*domain.Template{versionsTF, backendTF, providersTF, mainTF, variablesTF, outputsTF} {
		material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
		if err != nil {
			return err
		}

		p.AddJSON(material.FmtContents(), material.Path())
	}

	// Blobs stay byte-exact, preserving the #cloud-config header.
	for key, group := range data.Pinned {
		buf := bytes.NewBuffer(nil)
		if err := cloudInitYAML.Execute(buf, group); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute cloud-init template")
		}

		p.AddBlob(buf.Bytes(), "cloud-init", key+".yaml")
	}

	for key, group := range data.Pools {
		buf := bytes.NewBuffer(nil)
		if err := cloudInitYAML.Execute(buf, group); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute cloud-init template")
		}

		p.AddBlob(buf.Bytes(), "cloud-init", key+".yaml")
	}

	return nil
}

const planFile = "tfplan"

func (c *awsEcsTerraformCasting) Cast(ctx context.Context, config infrastructure.Casting, outputPath string, p *pourer.Pourer) error {
	c.logger.InfoContext(ctx, "Applying terraform", slog.String("release", config.Metadata.Name))

	root := filepath.Join(outputPath, p.Dir())

	if err := c.terraform(ctx, root, "init"); err != nil {
		return err
	}

	if err := c.terraform(ctx, root, "plan", "-out="+planFile); err != nil {
		return err
	}

	if err := c.terraform(ctx, root, "apply", planFile); err != nil {
		return err
	}

	c.logger.InfoContext(ctx, "Terraform applied successfully")

	return nil
}

func (c *awsEcsTerraformCasting) terraform(ctx context.Context, root, verb string, args ...string) error {
	argv := append([]string{"-chdir=" + root, verb}, args...)

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "terraform "+strings.Join(argv, " ")))

	// Stdout is foundry's own contract, so the tool's output goes to stderr.
	cmd := exec.CommandContext(ctx, "terraform", argv...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run terraform %s", verb)
	}

	return nil
}

func (c *awsEcsTerraformCasting) resources(config infrastructure.Casting) (*ecscontract.Resources, error) {
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
