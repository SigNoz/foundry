package v1alpha1

// InfrastructureProvider represents the cloud provider for infrastructure deployment.
type InfrastructureProvider string

const (
	InfrastructureProviderAWS   InfrastructureProvider = "aws"
	InfrastructureProviderGCP   InfrastructureProvider = "gcp"
	InfrastructureProviderAzure InfrastructureProvider = "azure"
)

// InfrastructureComputeType represents the compute type for infrastructure deployment.
type InfrastructureComputeType string

const (
	// AWS compute types.
	InfrastructureComputeTypeEC2 InfrastructureComputeType = "ec2"
	InfrastructureComputeTypeEKS InfrastructureComputeType = "eks"
	// GCP compute types.
	InfrastructureComputeTypeGCE InfrastructureComputeType = "gce"
	InfrastructureComputeTypeGKE InfrastructureComputeType = "gke"
	// Azure compute types.
	InfrastructureComputeTypeVM  InfrastructureComputeType = "vm"
	InfrastructureComputeTypeAKS InfrastructureComputeType = "aks"
)

// Infrastructure holds the configuration for infrastructure manifest generation (e.g., Terraform).
type Infrastructure struct {
	// Whether infrastructure manifest generation is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// The cloud provider to generate infrastructure manifests for (aws, gcp, azure)
	Provider InfrastructureProvider `json:"provider,omitempty" yaml:"provider,omitempty"`

	// The compute type for the provider (ec2, eks for AWS; gce, gke for GCP; vm, aks for Azure)
	ComputeType InfrastructureComputeType `json:"computeType,omitempty" yaml:"computeType,omitempty"`
}

// DefaultInfrastructure returns the default Infrastructure configuration.
func DefaultInfrastructure() Infrastructure {
	return Infrastructure{
		Enabled:     false,
		Provider:    InfrastructureProviderAWS,
		ComputeType: InfrastructureComputeTypeEC2,
	}
}
