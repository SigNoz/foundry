package installation

// Infrastructure is the installation's binding to the substrate it runs on. The
// consumer owns the binding because the two castings share no state: naming the
// substrate is what lets a casting derive the tag filter that finds its
// resources. Only a casting resolves it.
type Infrastructure struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty" pattern:"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" maxLength:"63" description:"Name of the infrastructure casting this installation runs on"`

	_ struct{} `additionalProperties:"false"`
}
