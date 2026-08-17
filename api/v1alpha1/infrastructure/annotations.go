package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// Substrate annotations for the ECS deployment of the Infrastructure Kind.
var (
	ECSRegion = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-region",
		Mode:        v1alpha1.ModeECS,
		Description: "AWS region the substrate is provisioned in; the declared subnet zones must belong to it.",
	}
)

func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		ECSRegion,
	}
}
