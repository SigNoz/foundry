package v1alpha1

// Annotation describes a single foundry annotation: its key, the default applied
// when the user omits it, the deployment mode it applies to, and a human
// description. It is a plain descriptor any Kind can build; the per-Kind catalogs
// (e.g. the installation package) hold the actual entries.
type Annotation struct {
	Key         string
	Default     string
	Mode        Mode
	Description string
}

// Resolve returns the user-set value for this annotation, or the default when it
// is absent. A nil map resolves to the default.
func (a Annotation) Resolve(annotations map[string]string) string {
	if value := annotations[a.Key]; value != "" {
		return value
	}
	return a.Default
}
