package v1alpha1

type Ingester struct {
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (i Ingester) MergeStatusIntoSpec() Ingester {
	return Ingester{
		Spec:   MergeStatusIntoSpec(i.Spec, i.Status),
		Status: i.Status,
	}
}
