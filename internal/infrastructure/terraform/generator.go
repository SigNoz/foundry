package terraform

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/infrastructure"
)

var _ infrastructure.Generator = (*Generator)(nil)

const infrastructureDir = "infrastructure"

// Generator generates Terraform manifests for infrastructure deployment.
type Generator struct {
	logger *slog.Logger
}

type templateData struct {
	installation.Casting
	Provider    v1alpha1.Platform
	ComputeType infrastructure.ComputeType
}

// New creates a new Terraform Generator.
func New(logger *slog.Logger) *Generator {
	return &Generator{
		logger: logger,
	}
}

// Generate creates Terraform manifests based on the casting configuration.
// The compute type is resolved automatically from the provider and deployment mode.
func (g *Generator) Generate(ctx context.Context, config installation.Casting) ([]domain.Material, error) {
	if !config.Spec.Infrastructure.Enabled {
		return nil, nil
	}

	provider, err := infrastructure.ResolveProvider(config.Spec.Deployment.Platform)
	if err != nil {
		return nil, err
	}
	computeType, err := infrastructure.ResolveComputeType(provider, config.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	g.logger.InfoContext(ctx, "generating terraform manifests",
		slog.String("provider", provider.String()),
		slog.String("computeType", computeType.String()),
	)

	data := templateData{
		Casting:     config,
		Provider:    provider,
		ComputeType: computeType,
	}

	mainTemplate, varsTemplate, outputsTemplate, err := g.templatesFor(provider, computeType)
	if err != nil {
		return nil, err
	}

	materials := make([]domain.Material, 0, 4)
	for _, item := range []struct {
		tmpl *domain.Template
		path string
	}{
		{mainTemplate, "main.tf.json"},
		{varsTemplate, "variables.tf.json"},
		{providersTFTemplate, "providers.tf.json"},
		{outputsTemplate, "outputs.tf.json"},
	} {
		m, err := item.tmpl.Render(data, filepath.Join(infrastructureDir, item.path))
		if err != nil {
			return nil, fmt.Errorf("failed to render %s: %w", item.path, err)
		}
		materials = append(materials, m)
	}

	return materials, nil
}

// Validate runs `terraform validate` against the manifests in poursPath/infrastructure.
func (g *Generator) Validate(ctx context.Context, poursPath string) error {
	infraDir := filepath.Join(poursPath, infrastructureDir)
	g.logger.InfoContext(ctx, "validating terraform manifests", slog.String("path", infraDir))

	cmd := exec.CommandContext(ctx, "terraform", "validate")
	cmd.Dir = infraDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform validate failed: %w\n%s", err, out)
	}
	return nil
}

// templatesFor returns the provider+compute-type specific templates.
func (g *Generator) templatesFor(provider v1alpha1.Platform, computeType infrastructure.ComputeType) (main, vars, outputs *domain.Template, err error) {
	switch provider {
	case v1alpha1.PlatformAWS:
		switch computeType {
		case infrastructure.ComputeTypeEC2:
			return awsEC2MainTFTemplate, awsEC2VariablesTFTemplate, awsEC2OutputsTFTemplate, nil
		case infrastructure.ComputeTypeEKS:
			return awsEKSMainTFTemplate, awsEKSVariablesTFTemplate, awsEKSOutputsTFTemplate, nil
		}
	case v1alpha1.PlatformGCP:
		switch computeType {
		case infrastructure.ComputeTypeGCE:
			return gcpGCEMainTFTemplate, gcpGCEVariablesTFTemplate, gcpGCEOutputsTFTemplate, nil
		case infrastructure.ComputeTypeGKE:
			return gcpGKEMainTFTemplate, gcpGKEVariablesTFTemplate, gcpGKEOutputsTFTemplate, nil
		}
	case v1alpha1.PlatformAzure:
		switch computeType {
		case infrastructure.ComputeTypeVM:
			return azureVMMainTFTemplate, azureVMVariablesTFTemplate, azureVMOutputsTFTemplate, nil
		case infrastructure.ComputeTypeAKS:
			return azureAKSMainTFTemplate, azureAKSVariablesTFTemplate, azureAKSOutputsTFTemplate, nil
		}
	}
	return nil, nil, nil, fmt.Errorf("unsupported provider %q / compute type %q combination", provider, computeType)
}
