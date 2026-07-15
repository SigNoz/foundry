package v1alpha1

// TypeCastingRef is the identity of a casting: which schema version and which
// name.
type TypeCastingRef struct {
	APIVersion string   `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty" enum:"v1alpha1" default:"v1alpha1" description:"API version of the referenced casting." example:"v1alpha1"`
	Name       string   `json:"name" yaml:"name" required:"true" nullable:"false" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"Name of the referenced casting."`
	_          struct{} `additionalProperties:"false"`
}
