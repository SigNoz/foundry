package v1alpha1

// Label describes a single foundry workload label: its key, the static value
// when one exists, and a human description.
type Label struct {
	Key         string
	Value       string
	Description string
}

// Ownership labels for workloads generated from any casting Kind. This is the
// single source for these keys, consumed by the casting templates.
var (
	LabelManagedBy = Label{
		Key:         "foundry.signoz.io/managed-by",
		Value:       "foundry",
		Description: "The tool that generated the workload.",
	}
	LabelKind = Label{
		Key:         "foundry.signoz.io/kind",
		Description: "The foundry casting Kind the workload belongs to.",
	}
	LabelName = Label{
		Key:         "foundry.signoz.io/name",
		Description: "The casting's metadata.name.",
	}
)

// Labels resolves the ownership labels from the casting's own fields.
// Castings stamp them onto every workload they generate.
func (meta CastingMeta) Labels() map[string]string {
	return map[string]string{
		LabelManagedBy.Key: LabelManagedBy.Value,
		LabelKind.Key:      meta.Kind.String(),
		LabelName.Key:      meta.Metadata.Name,
	}
}
