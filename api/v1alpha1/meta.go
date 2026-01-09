package v1alpha1

type TypeVersion struct {
	// API Version of the casting configuration schema.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
}

type TypeMetadata struct {
	// The name of this installation. This name can be used to identify the installation.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

type TypeCluster struct {
	// Number of replicas for the component
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Number of shards for the component
	Shards int `json:"shards,omitempty" yaml:"shards,omitempty"`
}

type TypeConfig struct {
	// Data contains the configuration data.
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
}

type TypeDeployment struct {
	// Mode in which the platform will run. Can be "binary", "docker", "kubernetes", etc.
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Platform on which the platform will run. Can be "aws", "gcp", "azure", etc.
	Platform string `json:"platform,omitempty" yaml:"platform,omitempty"`

	// OS on which the platform will run. Can be "linux", "windows", etc.
	OS string `json:"os,omitempty" yaml:"os,omitempty"`
}
