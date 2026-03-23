package terraform

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/infrastructure"
	"github.com/signoz/foundry/internal/types"
)

var _ infrastructure.Generator = (*Generator)(nil)

const infrastructureDir = "infrastructure"

// Generator generates Terraform manifests for infrastructure deployment.
type Generator struct {
	logger *slog.Logger
}

// New creates a new Terraform Generator.
func New(logger *slog.Logger) *Generator {
	return &Generator{
		logger: logger,
	}
}

// Generate creates Terraform manifests based on the casting configuration.
// The compute type is resolved automatically from the provider and deployment mode.
func (g *Generator) Generate(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	if !config.Spec.Infrastructure.Enabled {
		return nil, nil
	}

	provider := config.Spec.Infrastructure.Provider
	computeType, err := infrastructure.ResolveComputeType(provider, config.Spec.Deployment)
	if err != nil {
		return nil, err
	}

	g.logger.InfoContext(ctx, "generating terraform manifests",
		slog.String("provider", provider.String()),
		slog.String("computeType", computeType.String()),
	)

	mainTemplate, varsTemplate, outputsTemplate, err := g.templatesFor(provider, computeType)
	if err != nil {
		return nil, err
	}

	var materials []types.Material

	// main.tf
	mainBuf := bytes.NewBuffer(nil)
	if err := mainTemplate.Execute(mainBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute main.tf template: %w", err)
	}
	mainMaterial, err := types.NewHCLMaterial(mainBuf.Bytes(), filepath.Join(infrastructureDir, "main.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create main.tf material: %w", err)
	}
	materials = append(materials, mainMaterial)

	// variables.tf
	varsBuf := bytes.NewBuffer(nil)
	if err := varsTemplate.Execute(varsBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute variables.tf template: %w", err)
	}
	varsMaterial, err := types.NewHCLMaterial(varsBuf.Bytes(), filepath.Join(infrastructureDir, "variables.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create variables.tf material: %w", err)
	}
	materials = append(materials, varsMaterial)

	// providers.tf
	providersBuf := bytes.NewBuffer(nil)
	if err := providersTFTemplate.Execute(providersBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute providers.tf template: %w", err)
	}
	providersMaterial, err := types.NewHCLMaterial(providersBuf.Bytes(), filepath.Join(infrastructureDir, "providers.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create providers.tf material: %w", err)
	}
	materials = append(materials, providersMaterial)

	// outputs.tf
	outputsBuf := bytes.NewBuffer(nil)
	if err := outputsTemplate.Execute(outputsBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute outputs.tf template: %w", err)
	}
	outputsMaterial, err := types.NewHCLMaterial(outputsBuf.Bytes(), filepath.Join(infrastructureDir, "outputs.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create outputs.tf material: %w", err)
	}
	materials = append(materials, outputsMaterial)

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
func (g *Generator) templatesFor(provider v1alpha1.InfrastructureProvider, computeType infrastructure.ComputeType) (main, vars, outputs *types.Template, err error) {
	switch provider {
	case v1alpha1.InfrastructureProviderAWS:
		switch computeType {
		case infrastructure.ComputeTypeEC2:
			return awsEC2MainTFTemplate, awsEC2VariablesTFTemplate, awsEC2OutputsTFTemplate, nil
		case infrastructure.ComputeTypeEKS:
			return awsEKSMainTFTemplate, awsEKSVariablesTFTemplate, awsEKSOutputsTFTemplate, nil
		}
	case v1alpha1.InfrastructureProviderGCP:
		switch computeType {
		case infrastructure.ComputeTypeGCE:
			return gcpGCEMainTFTemplate, gcpGCEVariablesTFTemplate, gcpGCEOutputsTFTemplate, nil
		case infrastructure.ComputeTypeGKE:
			return gcpGKEMainTFTemplate, gcpGKEVariablesTFTemplate, gcpGKEOutputsTFTemplate, nil
		}
	case v1alpha1.InfrastructureProviderAzure:
		switch computeType {
		case infrastructure.ComputeTypeVM:
			return azureVMMainTFTemplate, azureVMVariablesTFTemplate, azureVMOutputsTFTemplate, nil
		case infrastructure.ComputeTypeAKS:
			return azureAKSMainTFTemplate, azureAKSVariablesTFTemplate, azureAKSOutputsTFTemplate, nil
		}
	}
	return nil, nil, nil, fmt.Errorf("unsupported provider %q / compute type %q combination", provider, computeType)
}
