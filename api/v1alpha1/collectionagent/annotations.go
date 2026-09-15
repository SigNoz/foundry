package collectionagent

import "github.com/signoz/foundry/api/v1alpha1"

// Cluster annotations for the ECS/EC2 deployment of the CollectionAgent Kind.
// The two roles die with the stack, so an absent one is created, not looked up.
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

var KubernetesNamespace = v1alpha1.Annotation{
	Key:         "foundry.signoz.io/kubernetes-namespace",
	Default:     "",
	Mode:        v1alpha1.ModeKubernetes,
	Description: "Namespace the collector is deployed into; unstated, the casting's metadata.name.",
}

var (
	HelmChart = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/kubernetes-helm-chart",
		Default:     "k8s-infra",
		Mode:        v1alpha1.ModeKubernetes,
		Description: "Chart to install: a name in the chart repository, a URL to a chart archive, or a local chart path.",
	}
	HelmChartRepoURL = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/kubernetes-helm-repo-url",
		Default:     "https://charts.signoz.io",
		Mode:        v1alpha1.ModeKubernetes,
		Description: "Chart repository URL the chart name is resolved against; unused when the chart states its own location.",
	}
	HelmChartVersion = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/kubernetes-helm-chart-version",
		Default:     "latest",
		Mode:        v1alpha1.ModeKubernetes,
		Description: "Helm chart version to install; `latest` installs the repository's latest chart.",
	}
)

func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		ECSRegion,
		ECSClusterARN,
		ECSTaskRoleARN,
		ECSTaskExecutionRoleARN,
		KubernetesNamespace,
		HelmChart,
		HelmChartRepoURL,
		HelmChartVersion,
	}
}
