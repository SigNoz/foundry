package terraformcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

const infrastructureDir = "infrastructure"

// TerraformGenerator generates Terraform manifests for infrastructure deployment.
type TerraformGenerator struct {
	logger *slog.Logger
}

// NewGenerator creates a new TerraformGenerator.
func NewGenerator(logger *slog.Logger) *TerraformGenerator {
	return &TerraformGenerator{
		logger: logger,
	}
}

// Generate creates Terraform manifests based on the casting configuration and infrastructure provider.
func (g *TerraformGenerator) Generate(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	if !config.Spec.Infrastructure.Enabled {
		return nil, nil
	}

	provider := config.Spec.Infrastructure.Provider
	computeType := config.Spec.Infrastructure.ComputeType
	g.logger.InfoContext(ctx, "Generating Terraform manifests",
		slog.String("provider", string(provider)),
		slog.String("computeType", string(computeType)),
	)

	var materials []types.Material

	// Get provider-specific templates
	mainTemplate, varsTemplate, outputsTemplate, err := g.getTemplatesForProvider(provider, computeType)
	if err != nil {
		return nil, err
	}

	// Generate main.tf
	mainBuf := bytes.NewBuffer(nil)
	if err := mainTemplate.Execute(mainBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute main.tf template: %w", err)
	}
	mainMaterial, err := types.NewHCLMaterial(mainBuf.Bytes(), filepath.Join(infrastructureDir, "main.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create main.tf material: %w", err)
	}
	materials = append(materials, mainMaterial)

	// Generate variables.tf
	varsBuf := bytes.NewBuffer(nil)
	if err := varsTemplate.Execute(varsBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute variables.tf template: %w", err)
	}
	varsMaterial, err := types.NewHCLMaterial(varsBuf.Bytes(), filepath.Join(infrastructureDir, "variables.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create variables.tf material: %w", err)
	}
	materials = append(materials, varsMaterial)

	// Generate providers.tf (common template)
	providersBuf := bytes.NewBuffer(nil)
	if err := providersTFTemplate.Execute(providersBuf, config); err != nil {
		return nil, fmt.Errorf("failed to execute providers.tf template: %w", err)
	}
	providersMaterial, err := types.NewHCLMaterial(providersBuf.Bytes(), filepath.Join(infrastructureDir, "providers.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to create providers.tf material: %w", err)
	}
	materials = append(materials, providersMaterial)

	// Generate outputs.tf
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

// getTemplatesForProvider returns the appropriate templates for the given infrastructure provider and compute type.
func (g *TerraformGenerator) getTemplatesForProvider(provider v1alpha1.InfrastructureProvider, computeType v1alpha1.InfrastructureComputeType) (main, vars, outputs *types.Template, err error) {
	switch provider {
	case v1alpha1.InfrastructureProviderAWS:
		switch computeType {
		case v1alpha1.InfrastructureComputeTypeEC2, "":
			return awsEC2MainTFTemplate, awsEC2VariablesTFTemplate, awsEC2OutputsTFTemplate, nil
		case v1alpha1.InfrastructureComputeTypeEKS:
			return awsEKSMainTFTemplate, awsEKSVariablesTFTemplate, awsEKSOutputsTFTemplate, nil
		default:
			return nil, nil, nil, fmt.Errorf("unsupported compute type %q for provider %q", computeType, provider)
		}
	case v1alpha1.InfrastructureProviderGCP:
		switch computeType {
		case v1alpha1.InfrastructureComputeTypeGCE, "":
			return gcpGCEMainTFTemplate, gcpGCEVariablesTFTemplate, gcpGCEOutputsTFTemplate, nil
		case v1alpha1.InfrastructureComputeTypeGKE:
			return gcpGKEMainTFTemplate, gcpGKEVariablesTFTemplate, gcpGKEOutputsTFTemplate, nil
		default:
			return nil, nil, nil, fmt.Errorf("unsupported compute type %q for provider %q", computeType, provider)
		}
	case v1alpha1.InfrastructureProviderAzure:
		switch computeType {
		case v1alpha1.InfrastructureComputeTypeVM, "":
			return azureVMMainTFTemplate, azureVMVariablesTFTemplate, azureVMOutputsTFTemplate, nil
		case v1alpha1.InfrastructureComputeTypeAKS:
			return azureAKSMainTFTemplate, azureAKSVariablesTFTemplate, azureAKSOutputsTFTemplate, nil
		default:
			return nil, nil, nil, fmt.Errorf("unsupported compute type %q for provider %q", computeType, provider)
		}
	default:
		return nil, nil, nil, fmt.Errorf("unsupported infrastructure provider: %s", provider)
	}
}
