package installation

// Infrastructure is the installation's binding to the infrastructure casting
// it runs on. The consumer owns the binding: it orders the casts
// (infrastructure first) and names the substrate for by-name lookups.
type Infrastructure struct {
	// Name of the infrastructure casting.
	Name string `json:"name,omitempty" yaml:"name,omitempty" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"Name of the infrastructure casting this installation runs on"`

	_ struct{} `additionalProperties:"false"`
}
