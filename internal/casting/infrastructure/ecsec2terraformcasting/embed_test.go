package ecsec2terraformcasting

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/molding/infrastructure/resourcemolding"
	"github.com/stretchr/testify/assert"
)

// A zone is per-account, so the fixture must state one.
const subnets = `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
`

func moldedCasting(t *testing.T) *infrastructure.Casting {
	t.Helper()

	config := infrastructure.Default()
	config.Spec.Resource.Spec.Config.Set(resourcemolding.ResourceConfigName, []byte(subnets))

	logger := slog.New(slog.DiscardHandler)
	assert.NoError(t, newEcsEc2TerraformMoldingEnricher().EnrichStatus(context.Background(), v1alpha1.MoldingKindResource, config))
	assert.NoError(t, resourcemolding.New(logger).MoldV1Alpha1(context.Background(), config))

	return config
}

func TestTemplates_RenderValidJSON(t *testing.T) {
	config := moldedCasting(t)

	data, err := newResources(*config)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		template *domain.Template
	}{
		{name: "ProvidersTemplate_RendersValidJSON", template: providersTFTemplate},
		{name: "MainTemplate_RendersValidJSON", template: mainTFTemplate},
		{name: "VariablesTemplate_RendersValidJSON", template: variablesTFTemplate},
		{name: "OutputsTemplate_RendersValidJSON", template: outputsTFTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := tt.template.Render(data, "out.tf.json")
			assert.NoError(t, err)
			assert.NotEmpty(t, material.FmtContents())
		})
	}
}

func TestMainTemplate_PinsPersistentAndPoolsEphemeral(t *testing.T) {
	config := moldedCasting(t)

	data, err := newResources(*config)
	assert.NoError(t, err)

	material, err := mainTFTemplate.Render(data, "out.tf.json")
	assert.NoError(t, err)

	contents := string(material.FmtContents())
	assert.Contains(t, contents, `"persistent-0"`)
	assert.Contains(t, contents, `"persistent-2"`)
	assert.Contains(t, contents, `"aws_ebs_volume"`)
	assert.Contains(t, contents, `"aws_volume_attachment"`)
	assert.Contains(t, contents, `"aws_autoscaling_group"`)
	assert.Contains(t, contents, "cloud-init/persistent.yaml")
	assert.Contains(t, contents, "cloud-init/ephemeral.yaml")
	assert.NotContains(t, contents, "aws_launch_template.persistent")
	assert.NotContains(t, contents, "aws_instance.ephemeral")
}

// The addresses below are the patch surface: renaming one breaks every stored
// spec.patches entry and, for resource labels, live state addresses.
func TestMainTemplate_FreezesThePatchSurface(t *testing.T) {
	config := moldedCasting(t)

	data, err := newResources(*config)
	assert.NoError(t, err)

	material, err := mainTFTemplate.Render(data, "out.tf.json")
	assert.NoError(t, err)

	main := map[string]any{}
	assert.NoError(t, json.Unmarshal(material.FmtContents(), &main))

	resources, _ := main["resource"].(map[string]any)

	expected := map[string][]string{
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

	assert.ElementsMatch(t, slices.Sorted(maps.Keys(expected)), slices.Sorted(maps.Keys(resources)))

	for resourceType, labels := range expected {
		instances, _ := resources[resourceType].(map[string]any)
		assert.ElementsMatch(t, labels, slices.Sorted(maps.Keys(instances)), "labels of %s", resourceType)
	}
}
