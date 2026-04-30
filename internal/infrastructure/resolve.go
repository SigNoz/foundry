package infrastructure

import (
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
)

// ResolveProvider maps a deployment platform to its InfrastructureProvider.
// Only the cloud platforms (aws, gcp, azure) and ECS map to a provider;
// managed platforms (render, coolify, railway) have no IaC backing.
func ResolveProvider(platform v1alpha1.Platform) (v1alpha1.InfrastructureProvider, error) {
	switch platform {
	case v1alpha1.PlatformAWS, v1alpha1.PlatformECS:
		return v1alpha1.InfrastructureProviderAWS, nil
	case v1alpha1.PlatformGCP:
		return v1alpha1.InfrastructureProviderGCP, nil
	case v1alpha1.PlatformAzure:
		return v1alpha1.InfrastructureProviderAzure, nil
	case v1alpha1.Platform{}:
		return v1alpha1.InfrastructureProvider{}, fmt.Errorf("no platform specified in deployment.platform: infrastructure generation requires aws, gcp, or azure")
	default:
		return v1alpha1.InfrastructureProvider{}, fmt.Errorf("unsupported platform for infrastructure generation: %q (must be aws, gcp, azure, or ecs)", platform)
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
	isKubernetes := deployment.Mode == v1alpha1.ModeKubernetes

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
