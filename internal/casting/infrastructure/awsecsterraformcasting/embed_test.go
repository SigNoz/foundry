package awsecsterraformcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	ecscontract "github.com/signoz/foundry/internal/contract/aws/ecs"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A zone is per-account, so the fixture must state one.
const subnets = `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
`

type mainDocument struct {
	Resource map[string]map[string]struct {
		UserData       string `json:"user_data"`
		UserDataBase64 string `json:"user_data_base64"`
	} `json:"resource"`
}

type cloudConfig struct {
	WriteFiles []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"write_files"`
}

func moldedCasting(t *testing.T) *infrastructure.Casting {
	t.Helper()

	config := infrastructure.Default()
	config.Metadata.Annotations = map[string]string{infrastructure.ECSRegion.Key: "us-east-1"}
	config.Spec.Resource.Spec.Config.Set(resourcemolding.ResourceConfigName, []byte(subnets))

	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)

	require.NoError(t, newAwsEcsTerraformMoldingEnricher().EnrichStatus(ctx, v1alpha1.MoldingKindResource, config))
	require.NoError(t, resourcemolding.New(logger).MoldV1Alpha1(ctx, config))
	require.NoError(t, config.MergeStatusIntoSpec())

	return config
}

func dataFor(t *testing.T, config *infrastructure.Casting) any {
	t.Helper()

	resources, err := New(slog.New(slog.DiscardHandler)).resources(*config)
	require.NoError(t, err)

	return struct {
		*ecscontract.Resources

		Region string
	}{resources, infrastructure.ECSRegion.Resolve(config.Metadata.Annotations)}
}

func render(t *testing.T, tmpl *domain.Template, data any) mainDocument {
	t.Helper()

	material, err := tmpl.Render(data, strings.TrimSuffix(tmpl.Name(), ".gotmpl"))
	require.NoError(t, err)

	var rendered mainDocument
	require.NoError(t, json.Unmarshal(material.FmtContents(), &rendered))

	return rendered
}

func TestTemplatesRender(t *testing.T) {
	data := dataFor(t, moldedCasting(t))

	for name, test := range map[string]struct {
		template *domain.Template
		pass     bool
	}{
		"Versions_Valid":  {versionsTF, true},
		"Backend_Valid":   {backendTF, true},
		"Providers_Valid": {providersTF, true},
		"Main_Valid":      {mainTF, true},
		"Variables_Valid": {variablesTF, true},
		"Outputs_Valid":   {outputsTF, true},
	} {
		t.Run(name, func(t *testing.T) {
			material, err := test.template.Render(data, strings.TrimSuffix(test.template.Name(), ".gotmpl"))

			if test.pass {
				require.NoError(t, err)
				assert.NotEmpty(t, material.FmtContents())

				return
			}

			require.Error(t, err)
		})
	}
}

func TestMainGroups(t *testing.T) {
	data := dataFor(t, moldedCasting(t))

	for _, test := range []struct {
		name                   string
		resourceType           string
		pass                   bool
		expectedLabels         []string
		expectedUserData       string
		expectedUserDataBase64 string
	}{
		{
			"Persistent_Valid", "aws_instance", true, []string{"persistent-0", "persistent-1", "persistent-2"},
			"", `${base64encode(templatefile("${path.module}/cloud-init/persistent.yaml", { cluster = aws_ecs_cluster.main.name }))}`,
		},
		{
			"Ephemeral_Valid", "aws_launch_template", true, []string{"ephemeral"},
			`${base64encode(templatefile("${path.module}/cloud-init/ephemeral.yaml", { cluster = aws_ecs_cluster.main.name }))}`, "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			material, err := mainTF.Render(data, strings.TrimSuffix(mainTF.Name(), ".gotmpl"))

			if test.pass {
				require.NoError(t, err)

				var rendered mainDocument
				require.NoError(t, json.Unmarshal(material.FmtContents(), &rendered))

				booted := rendered.Resource[test.resourceType]
				assert.ElementsMatch(t, test.expectedLabels, slices.Collect(maps.Keys(booted)))

				for label, resource := range booted {
					assert.Equal(t, test.expectedUserData, resource.UserData, label)
					assert.Equal(t, test.expectedUserDataBase64, resource.UserDataBase64, label)
				}

				return
			}

			require.Error(t, err)
		})
	}
}

// The addresses below are the patch surface: renaming one breaks every stored
// spec.patches entry and, for resource labels, live state addresses.
func TestMainPatchSurface(t *testing.T) {
	resources := render(t, mainTF, dataFor(t, moldedCasting(t))).Resource

	expectedLabels := map[string][]string{
		"aws_vpc":                             {"main"},
		"aws_subnet":                          {"private-a", "public-a"},
		"aws_internet_gateway":                {"main"},
		"aws_eip":                             {"private-a"},
		"aws_nat_gateway":                     {"private-a"},
		"aws_route_table":                     {"private-a", "public-a"},
		"aws_route":                           {"private-a", "public-a"},
		"aws_route_table_association":         {"private-a", "public-a"},
		"terraform_data":                      {"persistent"},
		"aws_instance":                        {"persistent-0", "persistent-1", "persistent-2"},
		"aws_ebs_volume":                      {"persistent-0", "persistent-1", "persistent-2"},
		"aws_volume_attachment":               {"persistent-0", "persistent-1", "persistent-2"},
		"aws_launch_template":                 {"ephemeral"},
		"aws_autoscaling_group":               {"ephemeral"},
		"aws_security_group":                  {"tasks"},
		"aws_vpc_security_group_ingress_rule": {"intra_cluster"},
		"aws_vpc_security_group_egress_rule":  {"all_outbound"},
		"aws_iam_role":                        {"node"},
		"aws_iam_instance_profile":            {"node"},
		"aws_iam_role_policy_attachment":      {"node"},
		"aws_ecs_cluster":                     {"main"},
	}

	assert.ElementsMatch(t, slices.Sorted(maps.Keys(expectedLabels)), slices.Sorted(maps.Keys(resources)))

	for resourceType, labels := range expectedLabels {
		assert.ElementsMatch(t, labels, slices.Sorted(maps.Keys(resources[resourceType])), "labels of %s", resourceType)
	}
}

// ${cluster} is left for terraform's templatefile, which main.tf.json calls
// with the cluster's name.
func TestCloudInitRender(t *testing.T) {
	resources, err := New(slog.New(slog.DiscardHandler)).resources(*moldedCasting(t))
	require.NoError(t, err)

	for _, test := range []struct {
		name           string
		group          any
		pass           bool
		expectedDropIn string
	}{
		{"Persistent_Valid", resources.Pinned[resourcemolding.GroupPersistent], true, "[Unit]\nRequiresMountsFor=/var/lib/foundry\n"},
		{"Ephemeral_Valid", resources.Pools[resourcemolding.GroupEphemeral], true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			err := cloudInitYAML.Execute(buf, test.group)

			if test.pass {
				require.NoError(t, err)

				header, _, _ := strings.Cut(buf.String(), "\n")
				assert.Equal(t, "#cloud-config", header)

				var rendered cloudConfig
				require.NoError(t, domain.UnmarshalYAML(buf.Bytes(), &rendered))

				files := map[string]string{}
				for _, file := range rendered.WriteFiles {
					files[file.Path] = file.Content
				}

				assert.Contains(t, files["/etc/ecs/ecs.config"], "ECS_CLUSTER=${cluster}\n")
				assert.Equal(t, test.expectedDropIn, files["/etc/systemd/system/ecs.service.d/10-foundry-data.conf"])

				return
			}

			require.Error(t, err)
		})
	}
}
