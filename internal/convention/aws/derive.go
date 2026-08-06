package aws

import (
	"github.com/signoz/foundry/internal/convention"
	"maps"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/errors"
)

// What the substrate's own roles and rules exist for. Each is a name segment.
var (
	purposeIntraCluster = convention.MustNewKey("intra-cluster")
	purposeAllOutbound  = convention.MustNewKey("all-outbound")
)

// Resources renders a declared substrate into every name and tag it stamps.
// Labels sit over the declaration's cloudLabels and under the derived tags. An
// operator cannot rename what a consumer matches on.
func Resources(s convention.Substrate, declaration *infrastructure.ResourceConfig, labels map[string]string) (*infrastructure.ResourceConfigResources, error) {
	base := map[string]string{}
	maps.Copy(base, declaration.CloudLabels)
	maps.Copy(base, labels)

	named := func(resource Resource) infrastructure.ResourceConfigResource {
		tags := maps.Clone(base)
		maps.Copy(tags, resource.Tags())

		return infrastructure.ResourceConfigResource{Name: resource.Name(), Tags: tags}
	}

	resources := &infrastructure.ResourceConfigResources{
		Cluster:         named(Cluster(s)),
		VPC:             named(VPC(s)),
		SecurityGroup:   named(SecurityGroup(s, RoleTask)),
		InstanceProfile: named(InstanceProfile(s, RoleNode)),
		SecurityGroupRules: map[string]infrastructure.ResourceConfigResource{
			purposeIntraCluster.String(): named(SecurityGroupRule(s, RoleTask, purposeIntraCluster)),
			purposeAllOutbound.String():  named(SecurityGroupRule(s, RoleTask, purposeAllOutbound)),
		},
		Roles:       map[string]infrastructure.ResourceConfigResource{},
		IgnoredTags: []string{Tag(convention.TagKeyIdentities)},
	}

	// An adopted network keeps its owner's name and tags, and gets no gateway.
	if id := declaration.Networking.NetworkID; id != "" {
		resources.VPC = infrastructure.ResourceConfigResource{ID: id}
	}

	// The node's own credential only. Without it the agent cannot register the
	// instance with the cluster. Workload identity belongs to the workload.
	resources.Roles[RoleNode.String()] = named(IAMRole(s, RoleNode))

	if err := networking(s, named, declaration, resources); err != nil {
		return nil, err
	}

	if err := instanceGroups(s, named, declaration, resources); err != nil {
		return nil, err
	}

	return resources, nil
}

func networking(s convention.Substrate,
	named func(Resource) infrastructure.ResourceConfigResource,
	declaration *infrastructure.ResourceConfig,
	resources *infrastructure.ResourceConfigResources,
) error {
	subnets := declaration.Networking.Subnets

	resources.Subnets = make(map[string]infrastructure.ResourceConfigResourceSubnet, len(subnets))
	resources.RouteTables = map[string]infrastructure.ResourceConfigResource{}
	resources.NATGateways = map[string]infrastructure.ResourceConfigResourceNATGateway{}

	// A gateway serves the private subnet it is keyed by and lives in a public
	// one in the same zone. Walked in key order to keep the choice stable.
	publicByZone := map[string]string{}

	for _, key := range slices.Sorted(maps.Keys(subnets)) {
		subnet := subnets[key]

		if !subnet.Type.IsPublic() || subnet.ID != "" {
			continue
		}

		if _, ok := publicByZone[subnet.Zone]; !ok {
			publicByZone[subnet.Zone] = key
		}
	}

	for _, key := range slices.Sorted(maps.Keys(subnets)) {
		subnet := subnets[key]

		reference, err := convention.NewKey(key)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive subnet %q", key)
		}

		// An adopted subnet keeps the operator's own routing.
		if subnet.ID != "" {
			resources.Subnets[key] = infrastructure.ResourceConfigResourceSubnet{ID: subnet.ID, Public: subnet.Type.IsPublic()}
			continue
		}

		resource := named(Subnet(s, reference, subnet.Type))

		resources.Subnets[key] = infrastructure.ResourceConfigResourceSubnet{
			Name:   resource.Name,
			Tags:   resource.Tags,
			Public: subnet.Type.IsPublic(),
		}

		resources.RouteTables[key] = named(RouteTable(s, reference))

		if subnet.Type.IsPublic() {
			// Derived here: the gateway exists only if a public subnet does.
			resources.InternetGateway = named(InternetGateway(s))
			continue
		}

		if subnet.Egress != "" {
			resources.NATGateways[key] = infrastructure.ResourceConfigResourceNATGateway{ID: subnet.Egress}
			continue
		}

		gateway := named(NATGateway(s, reference))
		address := named(ElasticIP(s, reference))

		resources.NATGateways[key] = infrastructure.ResourceConfigResourceNATGateway{
			Name:    gateway.Name,
			Tags:    gateway.Tags,
			Subnet:  publicByZone[subnet.Zone],
			Address: &address,
		}
	}

	return nil
}

func instanceGroups(s convention.Substrate,
	named func(Resource) infrastructure.ResourceConfigResource,
	declaration *infrastructure.ResourceConfig,
	resources *infrastructure.ResourceConfigResources,
) error {
	// A group that names no subnet is placed across every private one.
	placement := []string{}

	for _, key := range slices.Sorted(maps.Keys(declaration.Networking.Subnets)) {
		if !declaration.Networking.Subnets[key].Type.IsPublic() {
			placement = append(placement, key)
		}
	}

	resources.InstanceGroups = make(map[string]infrastructure.ResourceConfigResourceGroup, len(declaration.InstanceGroups))

	for _, key := range slices.Sorted(maps.Keys(declaration.InstanceGroups)) {
		declared := declaration.InstanceGroups[key]

		reference, err := convention.NewKey(key)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive instance group %q", key)
		}

		subnets := declared.Subnets

		if len(subnets) == 0 {
			subnets = placement
		}

		if len(subnets) == 0 {
			return errors.Newf(errors.TypeInvalidInput, "failed to derive instance group %q: there is no private subnet to place it in", key)
		}

		group := convention.NewNodeGroup(reference, declared.Storage)
		resolved := infrastructure.ResourceConfigResourceGroup{
			Storage:  declared.Storage,
			Selector: Filter(s.Select().WithStorage(declared.Storage)),
			Subnets:  subnets,
		}

		if !declared.Storage.IsPinned() {
			launchTemplate := named(LaunchTemplate(s, group))
			autoscalingGroup := named(AutoscalingGroup(s, group))

			resolved.LaunchTemplate = &launchTemplate
			resolved.AutoscalingGroup = &autoscalingGroup
			resources.InstanceGroups[key] = resolved

			continue
		}

		// Each node carries its volume. The two cannot land in different zones.
		nodes := 0

		if declared.MinSize != nil {
			nodes = *declared.MinSize
		}

		for ordinal := range nodes {
			volume := named(Volume(s, group, ordinal))
			node := named(Node(s, group, ordinal))

			resolved.Nodes = append(resolved.Nodes, infrastructure.ResourceConfigResourceNode{
				Name:    node.Name,
				Tags:    node.Tags,
				Ordinal: ordinal,
				Subnet:  subnets[ordinal%len(subnets)],
				Volume:  &volume,
			})
		}

		resources.InstanceGroups[key] = resolved
	}

	return nil
}
