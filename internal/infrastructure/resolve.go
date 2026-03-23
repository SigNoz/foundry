package infrastructure

import (
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
)

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
