package v1alpha1

// TypeResourceRef references a casting by identity. Path qualifies the file
// containing the referenced casting when it lives outside the declaring file;
// when empty, the reference resolves among the declaring file's own documents.
type TypeResourceRef struct {
	APIVersion string                 `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty" description:"API version of the referenced casting." example:"v1alpha1"`
	Kind       Kind                   `json:"kind" yaml:"kind" required:"true" description:"Kind of the referenced casting."`
	Name       string                 `json:"name" yaml:"name" required:"true" description:"Name of the referenced casting."`
	Path       string                 `json:"path,omitempty" yaml:"path,omitempty" description:"Relative path to the casting file containing the referenced casting, resolved against the declaring file's directory. Empty means the declaring file itself."`
	Status     *TypeResourceRefStatus `json:"status,omitempty" yaml:"status,omitempty" description:"Status of the reference. This is populated by foundry and written to the lock file."`
	_          struct{}               `additionalProperties:"false"`
}

// TypeResourceRefStatus records how a resource reference resolved at forge time.
type TypeResourceRefStatus struct {
	Conditions []TypeCondition `json:"conditions,omitempty" yaml:"conditions,omitempty" description:"Conditions observed while resolving the reference."`
	Path       string          `json:"path,omitempty" yaml:"path,omitempty" description:"Resolved source of the referenced casting."`
	Checksum   string          `json:"checksum,omitempty" yaml:"checksum,omitempty" description:"Checksum of the referenced casting source, as seen at forge."`
	_          struct{}        `additionalProperties:"false"`
}

// TypeCondition is a single observed condition, mirroring the kubernetes
// condition vocabulary.
type TypeCondition struct {
	Type    string   `json:"type" yaml:"type" required:"true" description:"Type of the condition." examples:"[\"ResolvedRefs\",\"Accepted\",\"Programmed\"]"`
	Status  string   `json:"status" yaml:"status" required:"true" description:"Status of the condition." examples:"[\"True\",\"False\"]"`
	Reason  string   `json:"reason,omitempty" yaml:"reason,omitempty" description:"Machine-readable reason for the condition."`
	Message string   `json:"message,omitempty" yaml:"message,omitempty" description:"Human-readable message for the condition."`
	_       struct{} `additionalProperties:"false"`
}
