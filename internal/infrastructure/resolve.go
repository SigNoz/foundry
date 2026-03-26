package infrastructure

import (
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
)

// ResolveProvider maps a deployment platform string to its InfrastructureProvider.
// Only aws, gcp, and azure are supported for infrastructure generation.
func ResolveProvider(platform string) (v1alpha1.InfrastructureProvider, error) {
	switch platform {
	case "aws":
		return v1alpha1.InfrastructureProviderAWS, nil
	case "gcp":
		return v1alpha1.InfrastructureProviderGCP, nil
	case "azure":
		return v1alpha1.InfrastructureProviderAzure, nil
	default:
		if platform == "" {
			return v1alpha1.InfrastructureProvider{}, fmt.Errorf("no platform specified in deployment.platform: infrastructure generation requires aws, gcp, or azure")
		}
		return v1alpha1.InfrastructureProvider{}, fmt.Errorf("unsupported platform for infrastructure generation: %q (must be aws, gcp, or azure)", platform)
	}
}

// ResolveComputeType derives the appropriate ComputeType from a cloud provider and
// deployment configuration. Users do not specify the compute type directly — foundry
// resolves it automatically using this matrix:
//
//	AWS   + kubernetes (any flavor) → EKS
//	AWS   + anything else           → EC2
//	GCP   + kubernetes (any flavor) → GKE
//	GCP   + anything else           → GCE
//	Azure + kubernetes (any flavor) → AKS
//	Azure + anything else           → VM
func ResolveComputeType(provider v1alpha1.InfrastructureProvider, deployment v1alpha1.TypeDeployment) (ComputeType, error) {
	isKubernetes := deployment.Mode == "kubernetes"

	switch provider {
	case v1alpha1.InfrastructureProviderAWS:
		if isKubernetes {
			return ComputeTypeEKS, nil
		}
		return ComputeTypeEC2, nil

	case v1alpha1.InfrastructureProviderGCP:
		if isKubernetes {
			return ComputeTypeGKE, nil
		}
		return ComputeTypeGCE, nil

	case v1alpha1.InfrastructureProviderAzure:
		if isKubernetes {
			return ComputeTypeAKS, nil
		}
		return ComputeTypeVM, nil

	default:
		return ComputeType{}, fmt.Errorf("unsupported infrastructure provider: %s", provider)
	}
}
