package awskubernetesserverlesskustomizecasting

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding/collectionagent/collectormolding"
	"github.com/signoz/foundry/internal/pourer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resources struct {
	Requests map[string]string `json:"requests"`
	Limits   map[string]string `json:"limits"`
}

type container struct {
	Name          string `json:"name"`
	RestartPolicy string `json:"restartPolicy"`
	VolumeMounts  []struct {
		Name      string `json:"name"`
		MountPath string `json:"mountPath"`
	} `json:"volumeMounts"`
	Resources resources `json:"resources"`
}

type podTemplate struct {
	Spec struct {
		Containers     []container `json:"containers"`
		InitContainers []container `json:"initContainers"`
		Volumes        []struct {
			Name      string `json:"name"`
			ConfigMap *struct {
				Name string `json:"name"`
			} `json:"configMap"`
		} `json:"volumes"`
	} `json:"spec"`
}

type manifest struct {
	Kind               string   `json:"kind"`
	Namespace          string   `json:"namespace"`
	Resources          []string `json:"resources"`
	ConfigMapGenerator []struct {
		Files []string `json:"files"`
	} `json:"configMapGenerator"`
	Rules []struct {
		Resources []string `json:"resources"`
	} `json:"rules"`
	Subjects []struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"subjects"`
	Spec struct {
		Mode     string `json:"mode"`
		Strategy struct {
			Type string `json:"type"`
		} `json:"strategy"`
		DeploymentUpdateStrategy struct {
			Type string `json:"type"`
		} `json:"deploymentUpdateStrategy"`
		Resources   resources   `json:"resources"`
		Template    podTemplate `json:"template"`
		JobTemplate struct {
			Spec struct {
				Template podTemplate `json:"template"`
			} `json:"spec"`
		} `json:"jobTemplate"`
	} `json:"spec"`
}

func castingWithKind(t *testing.T, kind collectionagent.CollectorKind) collectionagent.Casting {
	t.Helper()

	config := *collectionagent.Default()
	config.Spec.Collector.Kind = kind

	return config
}

// Fargate schedules no DaemonSet, and one collector scrapes every node.
func TestForge(t *testing.T) {
	operator := map[string]string{collectionagent.KubernetesCollectorController.Key: collectionagent.CollectorControllerOpenTelemetryOperator}
	deployment := []string{"kustomization.yaml", "namespace.yaml", "serviceaccount.yaml", "clusterrole.yaml", "clusterrolebinding.yaml", "service.yaml", "workload.yaml", "deployment.yaml"}
	deploymentOperator := []string{"kustomization.yaml", "namespace.yaml", "serviceaccount.yaml", "clusterrole.yaml", "clusterrolebinding.yaml", "opentelemetrycollector.yaml"}
	sidecar := []string{"kustomization.yaml", "role.yaml", "rolebinding.yaml", "patch.yaml", "patch-cronjob.yaml", "sidecar.yaml"}
	sidecarOperator := []string{"kustomization.yaml", "role.yaml", "rolebinding.yaml", "opentelemetrycollector.yaml"}

	for _, test := range []struct {
		name              string
		kind              collectionagent.CollectorKind
		annotations       map[string]string
		replicas          *int
		disabled          bool
		pass              bool
		expectedErrorType int
		expectedFiles     []string
	}{
		{"Deployment_Valid", collectionagent.CollectorKindDeployment, nil, nil, false, true, 0, deployment},
		{"DeploymentOperator_Valid", collectionagent.CollectorKindDeployment, operator, nil, false, true, 0, deploymentOperator},
		{"Agent_Invalid", collectionagent.CollectorKindAgent, nil, nil, false, false, foundryerrors.TypeUnsupported.ExitCode(), nil},
		{"SidecarDefault_Valid", collectionagent.CollectorKindSidecar, nil, nil, false, true, 0, sidecar},
		{"SidecarOperator_Valid", collectionagent.CollectorKindSidecar, operator, nil, false, true, 0, sidecarOperator},
		{"Replicas_Invalid", collectionagent.CollectorKindDeployment, nil, domain.NewIntPtr(2), false, false, foundryerrors.TypeUnsupported.ExitCode(), nil},
		{"SidecarReplicas_Invalid", collectionagent.CollectorKindSidecar, nil, domain.NewIntPtr(2), false, false, foundryerrors.TypeUnsupported.ExitCode(), nil},
		{"Disabled_Invalid", collectionagent.CollectorKindDeployment, nil, nil, true, false, foundryerrors.TypeUnsupported.ExitCode(), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := castingWithKind(t, test.kind)
			config.Metadata.Annotations = test.annotations
			config.Spec.Collector.Spec.Cluster.Replicas = test.replicas
			config.Spec.Collector.Spec.Enabled = domain.NewBoolPtr(!test.disabled)

			p := pourer.New("collectionagent")

			err := newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindCollector, &config)
			if err == nil {
				err = collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(context.Background(), &config)
			}

			if err == nil {
				err = config.MergeStatusIntoSpec()
			}

			if err == nil {
				err = New(slog.New(slog.DiscardHandler)).Forge(context.Background(), config, p)
			}

			if !test.pass {
				require.Error(t, err)
				assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))

				return
			}

			require.NoError(t, err)

			materials, err := p.Pour()
			require.NoError(t, err)

			var paths []string
			for _, material := range materials {
				paths = append(paths, material.Path())
			}

			dir := filepath.Join("collectionagent", filepath.Dir(test.kind.ConfigKey()))

			var expectedPaths []string
			for _, file := range test.expectedFiles {
				expectedPaths = append(expectedPaths, filepath.Join(dir, file))
			}

			assert.ElementsMatch(t, expectedPaths, paths)
		})
	}
}

func TestForgeCollectorController(t *testing.T) {
	for _, test := range []struct {
		name              string
		annotations       map[string]string
		pass              bool
		expectedErrorType int
	}{
		{"Unstated_Valid", nil, true, 0},
		{"DefaultStated_Valid", map[string]string{collectionagent.KubernetesCollectorController.Key: collectionagent.CollectorControllerDefault}, true, 0},
		{"OpenTelemetryOperator_Valid", map[string]string{collectionagent.KubernetesCollectorController.Key: collectionagent.CollectorControllerOpenTelemetryOperator}, true, 0},
		{"Unknown_Invalid", map[string]string{collectionagent.KubernetesCollectorController.Key: "argo"}, false, foundryerrors.TypeInvalidInput.ExitCode()},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := castingWithKind(t, collectionagent.CollectorKindDeployment)
			config.Metadata.Annotations = test.annotations

			err := New(slog.New(slog.DiscardHandler)).Forge(context.Background(), config, pourer.New("collectionagent"))

			if test.pass {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Equal(t, test.expectedErrorType, foundryerrors.ExitCode(err))
		})
	}
}

func TestSurface(t *testing.T) {
	operator := map[string]string{collectionagent.KubernetesCollectorController.Key: collectionagent.CollectorControllerOpenTelemetryOperator}
	shop := map[string]string{collectionagent.KubernetesNamespace.Key: "shop"}
	deploymentSize := map[string]string{"cpu": "250m", "memory": "512Mi"}
	sidecarSize := map[string]string{"cpu": "200m", "memory": "256Mi"}
	sidecarContainers := map[string]string{"signoz-collector-sidecar": "Always"}

	for _, test := range []struct {
		name                   string
		annotations            map[string]string
		template               *domain.Template
		pass                   bool
		expectedKind           string
		expectedNamespace      string
		expectedFiles          []string
		expectedManifests      []string
		expectedSubjects       []string
		expectedMountPath      string
		expectedConfigMap      string
		expectedStrategy       string
		expectedUpdateStrategy string
		expectedMode           string
		expectedResources      []string
		expectedSize           map[string]string
		expectedSpecSize       map[string]string
		expectedInitContainers map[string]string
	}{
		{name: "Kustomization_Valid", template: kustomizationTemplate, pass: true, expectedKind: "Kustomization", expectedNamespace: "signoz", expectedFiles: []string{"deployment.yaml"}},
		{name: "KustomizationOperator_Valid", annotations: operator, template: kustomizationTemplate, pass: true, expectedKind: "Kustomization", expectedNamespace: "signoz", expectedManifests: []string{"opentelemetrycollector.yaml"}},
		{name: "Workload_Valid", template: deploymentTemplate, pass: true, expectedKind: "Deployment", expectedMountPath: "/conf", expectedConfigMap: "signoz-collector-deployment", expectedStrategy: "Recreate", expectedSize: deploymentSize},
		{name: "OpenTelemetryCollector_Valid", annotations: operator, template: deploymentOpenTelemetryCollectorTemplate, pass: true, expectedKind: "OpenTelemetryCollector", expectedUpdateStrategy: "Recreate", expectedMode: "deployment", expectedSpecSize: deploymentSize},
		{name: "Clusterrole_Valid", template: deploymentClusterroleTemplate, pass: true, expectedKind: "ClusterRole", expectedResources: []string{"nodes/stats", "nodes/proxy", "persistentvolumeclaims", "persistentvolumes"}},
		{name: "SidecarKustomization_Valid", template: sidecarKustomizationTemplate, pass: true, expectedKind: "Kustomization", expectedFiles: []string{"sidecar.yaml"}, expectedManifests: []string{"role.yaml", "rolebinding.yaml"}},
		{name: "SidecarKustomizationNamespace_Valid", annotations: shop, template: sidecarKustomizationTemplate, pass: true, expectedKind: "Kustomization", expectedNamespace: "shop", expectedFiles: []string{"sidecar.yaml"}},
		{name: "Role_Valid", template: sidecarRoleTemplate, pass: true, expectedKind: "Role", expectedResources: []string{"pods", "replicasets", "jobs", "cronjobs"}},
		{name: "Rolebinding_Valid", template: sidecarRolebindingTemplate, pass: true, expectedKind: "RoleBinding", expectedSubjects: []string{"ServiceAccount/default"}},
		{name: "Patch_Valid", template: sidecarPatchTemplate, pass: true, expectedKind: "Deployment", expectedMountPath: "/conf", expectedConfigMap: "signoz-collector-sidecar", expectedSize: sidecarSize, expectedInitContainers: sidecarContainers},
		{name: "PatchCronjob_Valid", template: sidecarPatchCronjobTemplate, pass: true, expectedKind: "CronJob", expectedMountPath: "/conf", expectedConfigMap: "signoz-collector-sidecar", expectedSize: sidecarSize, expectedInitContainers: sidecarContainers},
		{name: "SidecarOpenTelemetryCollector_Valid", annotations: operator, template: sidecarOpenTelemetryCollectorTemplate, pass: true, expectedKind: "OpenTelemetryCollector", expectedMode: "sidecar", expectedSpecSize: sidecarSize},
		{name: "SidecarKustomizationOperator_Valid", annotations: operator, template: sidecarKustomizationTemplate, pass: true, expectedKind: "Kustomization", expectedManifests: []string{"role.yaml", "rolebinding.yaml", "opentelemetrycollector.yaml"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := castingWithKind(t, collectionagent.CollectorKindDeployment)
			config.Metadata.Annotations = test.annotations

			data := templateDataFor(config)

			body := bytes.NewBuffer(nil)
			require.NoError(t, sidecarPodtemplateTemplate.Execute(body, data))

			data.PodTemplate = body.String()

			material, err := test.template.Render(data, "out.yaml")
			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			var rendered manifest
			require.NoError(t, domain.UnmarshalYAML(material.FmtContents(), &rendered))

			assert.Equal(t, test.expectedKind, rendered.Kind)
			assert.Equal(t, test.expectedNamespace, rendered.Namespace)

			var files []string
			for _, generator := range rendered.ConfigMapGenerator {
				files = append(files, generator.Files...)
			}

			assert.Equal(t, test.expectedFiles, files)
			assert.Subset(t, rendered.Resources, test.expectedManifests)

			var subjects []string
			for _, subject := range rendered.Subjects {
				subjects = append(subjects, subject.Kind+"/"+subject.Name)
			}

			assert.Equal(t, test.expectedSubjects, subjects)

			var resources []string
			for _, rule := range rendered.Rules {
				resources = append(resources, rule.Resources...)
			}

			assert.Subset(t, resources, test.expectedResources)
			assert.Equal(t, test.expectedStrategy, rendered.Spec.Strategy.Type)
			assert.Equal(t, test.expectedUpdateStrategy, rendered.Spec.DeploymentUpdateStrategy.Type)
			assert.Equal(t, test.expectedMode, rendered.Spec.Mode)
			assert.Equal(t, test.expectedSpecSize, rendered.Spec.Resources.Requests)
			assert.Equal(t, test.expectedSpecSize, rendered.Spec.Resources.Limits)

			pod := rendered.Spec.Template
			if rendered.Kind == "CronJob" {
				pod = rendered.Spec.JobTemplate.Spec.Template
			}

			initContainers := map[string]string{}
			for _, container := range pod.Spec.InitContainers {
				initContainers[container.Name] = container.RestartPolicy
			}

			assert.Equal(t, len(test.expectedInitContainers), len(initContainers))
			for name, restartPolicy := range test.expectedInitContainers {
				assert.Equal(t, restartPolicy, initContainers[name], name)
			}

			mountPath := ""
			configMap := ""
			for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				assert.Equal(t, test.expectedSize, container.Resources.Requests)
				assert.Equal(t, test.expectedSize, container.Resources.Limits)

				for _, volume := range pod.Spec.Volumes {
					if volume.ConfigMap == nil {
						continue
					}

					for _, mount := range container.VolumeMounts {
						if mount.Name == volume.Name {
							mountPath = mount.MountPath
							configMap = volume.ConfigMap.Name
						}
					}
				}
			}

			assert.Equal(t, test.expectedMountPath, mountPath)
			assert.Equal(t, test.expectedConfigMap, configMap)
		})
	}
}

// What EKS Fargate adds rides in the enricher and has to survive the molding's merge.
func TestMoldedConfig(t *testing.T) {
	deploymentContains := map[string]string{
		"service.extensions":                  "k8s_observer",
		"service.pipelines.metrics.receivers": "receiver_creator",
	}
	deploymentValues := map[string]string{
		"extensions.k8s_observer.observe_nodes":                                             "true",
		"extensions.k8s_observer.observe_pods":                                              "false",
		"receivers.receiver_creator.receivers.kubeletstats.rule":                            `type == "k8s.node" && labels["eks.amazonaws.com/compute-type"] == "fargate"`,
		"receivers.receiver_creator.receivers.kubeletstats.config.auth_type":                "kubeConfig",
		"receivers.receiver_creator.receivers.kubeletstats.config.endpoint":                 "`name`",
		"receivers.receiver_creator.receivers.kubeletstats.config.node":                     "`name`",
		"receivers.receiver_creator.receivers.kubeletstats.config.k8s_api_config.auth_type": "serviceAccount",
		"processors.memory_limiter.limit_percentage":                                        "80",
		"processors.memory_limiter.spike_limit_percentage":                                  "20",
	}
	deploymentAbsent := []string{
		"receivers.receiver_creator.receivers.kubeletstats.config.insecure_skip_verify",
		"receivers.hostmetrics",
		"receivers.filelog",
		"receivers.kubeletstats",
		"processors.memory_limiter.limit_mib",
		"processors.memory_limiter.spike_limit_mib",
	}

	for _, test := range []struct {
		name               string
		kind               collectionagent.CollectorKind
		env                map[string]string
		pass               bool
		expectedAttributes map[string]string
		expectedContains   map[string]string
		expectedValues     map[string]string
		expectedAbsent     []string
	}{
		{
			name: "ClusterName_Valid",
			kind: collectionagent.CollectorKindDeployment,
			env:  map[string]string{"K8S_CLUSTER_NAME": "production", "SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318"},
			pass: true,
			expectedAttributes: map[string]string{
				"cloud.provider":   "aws",
				"cloud.platform":   "aws_eks",
				"k8s.cluster.name": "${env:K8S_CLUSTER_NAME}",
			},
			expectedContains: deploymentContains,
			expectedValues:   deploymentValues,
			expectedAbsent:   deploymentAbsent,
		},
		{
			name:               "NoClusterName_Valid",
			kind:               collectionagent.CollectorKindDeployment,
			env:                map[string]string{},
			pass:               true,
			expectedAttributes: map[string]string{"cloud.provider": "aws", "cloud.platform": "aws_eks"},
			expectedContains:   deploymentContains,
			expectedValues:     deploymentValues,
			expectedAbsent:     deploymentAbsent,
		},
		{
			name: "Sidecar_Valid",
			kind: collectionagent.CollectorKindSidecar,
			env:  map[string]string{"K8S_CLUSTER_NAME": "production", "SIGNOZ_INGESTION_ENDPOINT": "http://signoz:4318"},
			pass: true,
			expectedAttributes: map[string]string{
				"k8s.pod.name":       "${env:K8S_POD_NAME}",
				"k8s.pod.uid":        "${env:K8S_POD_UID}",
				"k8s.namespace.name": "${env:K8S_NAMESPACE}",
				"k8s.node.name":      "${env:K8S_NODE_NAME}",
				"cloud.provider":     "aws",
				"cloud.platform":     "aws_eks",
				"k8s.cluster.name":   "${env:K8S_CLUSTER_NAME}",
			},
			expectedValues: map[string]string{
				"processors.k8sattributes.filter.namespace":                 "${env:K8S_NAMESPACE}",
				"processors.k8sattributes.pod_association.#":                "1",
				"processors.k8sattributes.pod_association.0.sources.#":      "1",
				"processors.k8sattributes.pod_association.0.sources.0.from": "resource_attribute",
				"processors.k8sattributes.pod_association.0.sources.0.name": "k8s.pod.uid",
			},
			expectedAbsent: []string{
				"receivers.kubeletstats",
				"receivers.k8s_cluster",
				"receivers.k8s_events",
				"receivers.receiver_creator",
				"extensions.k8s_observer",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			config := castingWithKind(t, test.kind)
			config.Spec.Collector.Spec.Env = test.env

			err := newAwsKubernetesServerlessKustomizeMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindCollector, &config)
			if err == nil {
				err = collectormolding.New(slog.New(slog.DiscardHandler)).MoldV1Alpha1(ctx, &config)
			}

			if err == nil {
				err = config.MergeStatusIntoSpec()
			}

			if !test.pass {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)

			contents := config.Spec.Collector.Spec.Config.Data[test.kind.ConfigKey()]
			require.NotEmpty(t, contents)

			merged := domain.MustNewYAMLMaterial([]byte(contents), "deployment.yaml")

			for _, pipeline := range []string{"traces", "metrics", "logs"} {
				processors, err := merged.GetStringSlice("service.pipelines." + pipeline + ".processors")
				require.NoError(t, err)
				assert.Equal(t, []string{"memory_limiter", "resourcedetection", "resource/identity", "k8sattributes", "batch"}, processors, pipeline)
			}

			raw, err := merged.GetBytes("processors.resource/identity.attributes")
			require.NoError(t, err)

			var identity []struct {
				Key    string `json:"key"`
				Value  string `json:"value"`
				Action string `json:"action"`
			}
			require.NoError(t, json.Unmarshal(raw, &identity))

			attributes := map[string]string{}
			for _, attribute := range identity {
				assert.Equal(t, "insert", attribute.Action, attribute.Key)

				attributes[attribute.Key] = attribute.Value
			}

			assert.Equal(t, test.expectedAttributes, attributes)

			detectors, err := merged.GetStringSlice("processors.resourcedetection.detectors")
			require.NoError(t, err)
			assert.Equal(t, []string{"env"}, detectors)

			for path, expected := range test.expectedContains {
				actual, err := merged.GetStringSlice(path)
				require.NoError(t, err)
				assert.Contains(t, actual, expected, path)
			}

			for path, expected := range test.expectedValues {
				actual, err := merged.GetBytes(path)
				require.NoError(t, err, path)
				assert.Equal(t, expected, string(actual), path)
			}

			for _, path := range test.expectedAbsent {
				_, err := merged.GetBytes(path)
				assert.Error(t, err, path)
			}
		})
	}
}
