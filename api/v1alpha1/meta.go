package v1alpha1

type TypeVersion struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion" required:"true" nullable:"false" enum:"v1alpha1" description:"API Version of the configuration schema." default:"v1alpha1" example:"v1alpha1"`
}

type TypeMetadata struct {
	Name        string            `json:"name" yaml:"name" required:"true" nullable:"false" description:"The name of this installation. This name is used to identify the installation." default:"signoz" example:"signoz"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty" nullable:"true" description:"Annotations is an unstructured key-value map for arbitrary metadata. Can be used to specify deployment-specific settings."`
}

type TypeCluster struct {
	Replicas *int `json:"replicas,omitempty" yaml:"replicas,omitempty" description:"Number of replicas for the molding." example:"1"`
	Shards   *int `json:"shards,omitempty" yaml:"shards,omitempty" description:"Number of shards for the molding" example:"1"`
}

type TypeConfig struct {
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty" description:"Configuration data as key-value pairs."`
}

type TypeDeployment struct {
	// Platform: Provider where an installation runs on using various cloud vendors
	// Example values: aws|gcp|azure|digitalocean|railway
	Platform string `json:"platform,omitempty" yaml:"platform,omitempty" description:"Provider where an installation runs on" examples:"[\"aws\",\"gcp\",\"azure\",\"digitalocean\",\"railway\",\"docker\",\"linux\"]"`

	// Mode: Type of installation method that we support, currently identifies the engine or technology behind a deployment
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty" description:"Type of installation method" examples:"[\"binary\",\"docker\",\"kubernetes\",\"helm\",\"nomad\",\"windows\",\"systemctl\"]"`

	// Flavor: Defines the flavor of mode for the deployment, allows the user the pattern to deploy on
	Flavor string `json:"flavor,omitempty" yaml:"flavor,omitempty" description:"Flavor of mode for the deployment" examples:"[\"compose\",\"swarm\",\"helmfile\",\"helm\",\"kustomize\",\"binary\",\"rpm\",\"deb\",\"chocolatey\"]"`
}
