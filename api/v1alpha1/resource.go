package v1alpha1

// TypeResourceRef references a casting by identity. References resolve among
// the declaring file's own documents.
type TypeResourceRef struct {
	APIVersion string   `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty" enum:"v1alpha1" description:"API version of the referenced casting." example:"v1alpha1"`
	Kind       Kind     `json:"kind" yaml:"kind" required:"true" description:"Kind of the referenced casting."`
	Name       string   `json:"name" yaml:"name" required:"true" nullable:"false" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"Name of the referenced casting."`
	_          struct{} `additionalProperties:"false"`
}
