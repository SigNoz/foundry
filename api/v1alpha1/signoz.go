package v1alpha1

type SigNoz struct {
	Spec MoldingSpec `json:"spec,omitempty" yaml:"spec,omitempty"`

	Status MoldingStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

func (s SigNoz) MergeStatusIntoSpec() SigNoz {
	return SigNoz{
		Spec:   MergeStatusIntoSpec(s.Spec, s.Status),
		Status: s.Status,
	}
}
