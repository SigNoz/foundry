package resourcemolding

import (
	"context"
	"github.com/signoz/foundry/internal/convention"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
)

// What a casting contributes and an operator states between them: the baseline
// carries no zone and no machine type, because neither is the kind's to know.
const (
	networking = `networking:
  subnets:
    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}
    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}
`
	machineTypes = `instanceGroups:
  persistent: {machineType: m5.large}
  ephemeral: {machineType: c5.large}
`
	ephemeralMachineType = `instanceGroups:
  ephemeral: {machineType: c5.large}
`
)

// mold merges the given documents into one declaration, runs the molding over
// a config whose spec carries it, and returns what settled.
func mold(t *testing.T, documents ...string) (string, error) {
	t.Helper()

	config := infrastructure.Default()
	config.Metadata.Name = "foundry"

	declaration := ""
	for _, document := range documents {
		if declaration == "" {
			declaration = document
			continue
		}

		merged, err := domain.MergeYAML(declaration, document)
		if err != nil {
			t.Fatal(err)
		}

		declaration = merged
	}

	if declaration != "" {
		config.Spec.Resource.Spec.Config.Set(ResourceConfigName, []byte(declaration))
	}

	err := New(slog.New(slog.DiscardHandler), derive).MoldV1Alpha1(context.Background(), config)

	return config.Spec.Resource.Status.Config.Data[ResourceConfigName], err
}

// derive stands in for a provider's topology walk. The molding is neutral: it
// settles the declaration and hands it over, and what a provider makes of it is
// tested beside that provider.
func derive(_ convention.Substrate, declaration *infrastructure.ResourceConfig, _ map[string]string) (*infrastructure.ResourceConfigResources, error) {
	resources := &infrastructure.ResourceConfigResources{
		InstanceGroups: map[string]infrastructure.ResourceConfigResourceGroup{},
	}

	for key, group := range declaration.InstanceGroups {
		resolved := infrastructure.ResourceConfigResourceGroup{Storage: group.Storage}

		if group.Storage.IsPinned() && group.MinSize != nil {
			resolved.Nodes = make([]infrastructure.ResourceConfigResourceNode, *group.MinSize)
		}

		resources.InstanceGroups[key] = resolved
	}

	return resources, nil
}

func TestMoldV1Alpha1(t *testing.T) {
	doc, err := mold(t, networking, machineTypes)
	assert.NoError(t, err)

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(doc), &got))

	// One baseline for every substrate: the default installation shape.
	assert.Equal(t, map[string]infrastructure.ResourceConfigInstanceGroup{
		GroupPersistent: {
			Storage:     v1alpha1.StorageClassPersistent,
			MachineType: "m5.large",
			MinSize:     v1alpha1.IntPtr(3),
			MaxSize:     v1alpha1.IntPtr(3),
			RootVolume:  infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
			DataVolume:  &infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(50)},
		},
		GroupEphemeral: {
			Storage:     v1alpha1.StorageClassEphemeral,
			MachineType: "c5.large",
			MinSize:     v1alpha1.IntPtr(1),
			MaxSize:     v1alpha1.IntPtr(1),
			RootVolume:  infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
		},
	}, got.InstanceGroups)
}

// A substrate that keeps nothing drops the persistent group; the null deletes
// the key under RFC 7386.
func TestMoldV1Alpha1_StatelessSubstrate(t *testing.T) {
	doc, err := mold(t, networking+"instanceGroups:\n  persistent: null\n  ephemeral: {machineType: c5.large}\n")
	assert.NoError(t, err)

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(doc), &got))
	assert.Len(t, got.InstanceGroups, 1)
	assert.Contains(t, got.InstanceGroups, GroupEphemeral)
	assert.Empty(t, got.Resources.InstanceGroups[GroupPersistent].Nodes)
}

// A declaration with no subnets cannot be completed by anything but the
// operator, so it fails rather than guessing a zone.
func TestMoldV1Alpha1_BaselineAloneIsIncomplete(t *testing.T) {
	_, err := mold(t)

	assert.Error(t, err)
}

func TestMoldV1Alpha1_PreservesEnricherContributions(t *testing.T) {
	config := infrastructure.Default()
	config.Metadata.Name = "foundry"
	config.Spec.Resource.Status.Config.Set(ResourceConfigName, []byte(networking+`instanceGroups:
  persistent:
    machineType: m5.large
    minSize: 4
    maxSize: 4
    spotAllocation: lowest-price
  ephemeral:
    machineType: c5.large
    minSize: 2
    maxSize: 2
`))

	err := New(slog.New(slog.DiscardHandler), derive).MoldV1Alpha1(context.Background(), config)
	assert.NoError(t, err)

	doc := config.Spec.Resource.Status.Config.Data[ResourceConfigName]

	// Casting-specific keys survive the merge untouched.
	assert.Contains(t, doc, "spotAllocation")

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(doc), &got))

	// Groups are keyed, so the contribution states only the fields it changes
	// and the baseline's others survive under plain document merge, with no
	// list strategy.
	assert.Len(t, got.InstanceGroups, 2)

	persistent := got.InstanceGroups[GroupPersistent]
	assert.Equal(t, v1alpha1.IntPtr(4), persistent.MinSize)
	assert.Equal(t, v1alpha1.IntPtr(50), persistent.DataVolume.Size)
	assert.Len(t, got.Resources.InstanceGroups[GroupPersistent].Nodes, 4)

	ephemeral := got.InstanceGroups[GroupEphemeral]
	assert.Equal(t, v1alpha1.IntPtr(2), ephemeral.MinSize)
	assert.Equal(t, v1alpha1.IntPtr(30), ephemeral.RootVolume.Size)
}

// The operator's spec beats the casting's contribution wherever the two
// disagree.
func TestMoldV1Alpha1_SpecBeatsContribution(t *testing.T) {
	config := infrastructure.Default()
	config.Metadata.Name = "foundry"
	config.Spec.Resource.Status.Config.Set(ResourceConfigName, []byte(machineTypes))
	config.Spec.Resource.Spec.Config.Set(ResourceConfigName, []byte(networking+`instanceGroups:
  persistent: {machineType: r5.xlarge}
`))

	err := New(slog.New(slog.DiscardHandler), derive).MoldV1Alpha1(context.Background(), config)
	assert.NoError(t, err)

	got := infrastructure.ResourceConfig{}
	assert.NoError(t, domain.UnmarshalYAML([]byte(config.Spec.Resource.Status.Config.Data[ResourceConfigName]), &got))
	assert.Equal(t, "r5.xlarge", got.InstanceGroups[GroupPersistent].MachineType)
	assert.Equal(t, "c5.large", got.InstanceGroups[GroupEphemeral].MachineType)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		documents []string
		pass      bool
	}{
		{
			// A partial override is the point of keying: the baseline supplies
			// everything the declaration does not mention.
			name:      "PartialOverride_Valid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  persistent: {minSize: 1, maxSize: 1}\n"},
			pass:      true,
		},
		{
			name:      "NoSubnets_Invalid",
			documents: []string{machineTypes},
			pass:      false,
		},
		{
			name:      "SubnetWithoutZone_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-a: {type: private, cidr: 10.0.0.0/19}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "SubnetWithoutType_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-a: {zone: us-east-1a, cidr: 10.0.0.0/19}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "SubnetWithBadCIDR_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "SubnetKeyNotAName_Invalid",
			documents: []string{"networking:\n  subnets:\n    Private_A: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "OnlyPublicSubnets_Invalid",
			documents: []string{"networking:\n  subnets:\n    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}\n", machineTypes},
			pass:      false,
		},
		{
			// Nothing can place the gateway the private subnet needs.
			name:      "PrivateSubnetWithNoPublicInItsZone_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-b: {type: private, zone: us-east-1b, cidr: 10.0.32.0/19}\n    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22}\n", machineTypes},
			pass:      false,
		},
		{
			// An adopted egress path is one the operator already routes through.
			name:      "PrivateSubnetWithAdoptedEgress_Valid",
			documents: []string{"networking:\n  subnets:\n    private-b: {type: private, zone: us-east-1b, cidr: 10.0.32.0/19, egress: nat-0a1b2c3d}\n", machineTypes},
			pass:      true,
		},
		{
			name:      "PublicSubnetWithEgress_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}\n    public-a: {type: public, zone: us-east-1a, cidr: 10.0.96.0/22, egress: nat-0a1b2c3d}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "AdoptedNetworkWithAdoptedSubnets_Valid",
			documents: []string{"networking:\n  networkID: vpc-0a1b2c3d\n  subnets:\n    private-a: {type: private, zone: us-east-1a, id: subnet-0a1b2c3d}\n", machineTypes},
			pass:      true,
		},
		{
			name:      "AdoptedNetworkWithCreatedSubnet_Invalid",
			documents: []string{"networking:\n  networkID: vpc-0a1b2c3d\n  subnets:\n    private-a: {type: private, zone: us-east-1a, cidr: 10.0.0.0/19}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "AdoptedSubnetInCreatedNetwork_Invalid",
			documents: []string{"networking:\n  subnets:\n    private-a: {type: private, zone: us-east-1a, id: subnet-0a1b2c3d}\n", machineTypes},
			pass:      false,
		},
		{
			name:      "GroupWithoutMachineType_Invalid",
			documents: []string{networking},
			pass:      false,
		},
		{
			name:      "PinnedGroupWithUnequalBounds_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  persistent: {minSize: 3, maxSize: 5}\n"},
			pass:      false,
		},
		{
			name:      "GroupWithMaxBelowMin_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  ephemeral: {minSize: 4, maxSize: 2}\n"},
			pass:      false,
		},
		{
			// A null deletes the key under RFC 7386, so this removes the
			// baseline's data volume from a class that requires one. It is one
			// document because a null merged onto a document that never had
			// the key is dropped, not carried forward.
			name:      "PersistentWithoutDataVolume_Invalid",
			documents: []string{networking + "instanceGroups:\n  persistent: {machineType: m5.large, dataVolume: null}\n  ephemeral: {machineType: c5.large}\n"},
			pass:      false,
		},
		{
			name:      "EphemeralWithDataVolume_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  ephemeral: {dataVolume: {size: 20}}\n"},
			pass:      false,
		},
		{
			name:      "GroupKeyNotAName_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  Hot_Pool: {storage: ephemeral, machineType: c5.large, minSize: 1, maxSize: 1, rootVolume: {size: 30}}\n"},
			pass:      false,
		},
		{
			name:      "GroupPlacedInUndeclaredSubnet_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  persistent: {subnets: [private-z]}\n"},
			pass:      false,
		},
		{
			name:      "GroupPlacedInPublicSubnet_Invalid",
			documents: []string{networking, machineTypes, "instanceGroups:\n  persistent: {subnets: [public-a]}\n"},
			pass:      false,
		},
		{
			// A stated name and the tag derived beside it would disagree, and
			// the consuming casting would match nothing.
			name:      "StatedResources_Invalid",
			documents: []string{networking, machineTypes, "resources:\n  vpc: {name: my-vpc}\n"},
			pass:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mold(t, tt.documents...)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
