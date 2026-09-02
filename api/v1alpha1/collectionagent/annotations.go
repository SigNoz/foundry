package collectionagent

import "github.com/signoz/foundry/api/v1alpha1"

// Cluster annotations for the ECS/EC2 deployment of the CollectionAgent Kind.
// The agent is node-scoped and binds the instance's own network, so it names
// only the region and the cluster it runs a task on every instance of. The two
// roles hold no data and die with the stack, so an absent one is created rather
// than looked up.
var (
	ECSRegion = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-region",
		Mode:        v1alpha1.ModeEC2,
		Description: "AWS region holding the cluster.",
	}
	ECSClusterARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-cluster-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "ARN of the ECS cluster to run the agent on.",
	}
	ECSTaskRoleARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-task-role-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "IAM role ARN assumed by the agent task; needs read access to AWS AppConfig. Created when absent.",
	}
	ECSTaskExecutionRoleARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-task-execution-role-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "IAM role ARN the ECS agent assumes to pull images and start tasks. Created when absent.",
	}
)

// Annotations returns the CollectionAgent annotation catalog.
func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		ECSRegion,
		ECSClusterARN,
		ECSTaskRoleARN,
		ECSTaskExecutionRoleARN,
	}
}
