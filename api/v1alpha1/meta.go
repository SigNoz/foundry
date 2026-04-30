package v1alpha1

type TypeVersion struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion" required:"true" nullable:"false" enum:"v1alpha1" description:"API Version of the configuration schema." default:"v1alpha1" example:"v1alpha1"`
}

type TypeMetadata struct {
	Name        string            `json:"name" yaml:"name" required:"true" nullable:"false" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"The name of this installation. This name is used to identify the installation." default:"signoz" example:"signoz"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty" nullable:"true" description:"Annotations is an unstructured key-value map for arbitrary metadata. Can be used to specify deployment-specific settings."`
	_           struct{}          `additionalProperties:"false"`
}

type TypeCluster struct {
	Replicas *int     `json:"replicas,omitempty" yaml:"replicas,omitempty" minimum:"0" description:"Number of replicas for the molding." example:"1"`
	Shards   *int     `json:"shards,omitempty" yaml:"shards,omitempty" minimum:"1" description:"Number of shards for the molding" example:"1"`
	_        struct{} `additionalProperties:"false"`
}

type TypeConfig struct {
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty" description:"Configuration data as key-value pairs."`
	_    struct{}          `additionalProperties:"false"`
}

type TypeDeployment struct {
	// Platform: cloud or hosting provider where an installation runs.
	Platform Platform `json:"platform,omitzero" yaml:"platform,omitempty" description:"Provider where an installation runs on"`

	// Mode: type of installation method (engine or technology behind the deployment).
	Mode Mode `json:"mode,omitzero" yaml:"mode,omitempty" description:"Type of installation method"`

	// Flavor: variant of the mode for the deployment.
	Flavor Flavor `json:"flavor,omitzero" yaml:"flavor,omitempty" description:"Flavor of mode for the deployment"`

	_ struct{} `additionalProperties:"false"`
}
