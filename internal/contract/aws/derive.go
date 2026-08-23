package aws

import (
	"maps"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/errors"
)

// Stamper renders a descriptor into the resource it resolves to. Tags layer
// the declaration's cloudLabels first, then the labels, then the descriptor's
// own, so an operator cannot rename what a consumer matches on.
func Stamper(declaration *infrastructure.ResourceConfig, labels map[string]string) func(Descriptor) Resource {
	base := map[string]string{}
	maps.Copy(base, declaration.CloudLabels)
	maps.Copy(base, labels)

	return func(resource Descriptor) Resource {
		tags := maps.Clone(base)
		maps.Copy(tags, resource.Tags())

		return Resource{Name: resource.Name(), Tags: tags}
	}
}

// Networking derives the network every substrate shares: subnets, route tables
// and gateways. An adopted network keeps its owner's name and tags, and gets no
// gateway.
func Networking(s contract.Substrate,
	named func(Descriptor) Resource,
	declaration *infrastructure.ResourceConfig,
) (*NetworkResources, error) {
	subnets := declaration.Networking.Subnets

	network := &NetworkResources{
		VPC:         named(VPC(s)),
		Subnets:     make(map[string]SubnetResource, len(subnets)),
		RouteTables: map[string]Resource{},
		NATGateways: map[string]NATGatewayResource{},
	}

	if id := declaration.Networking.NetworkID; id != "" {
		network.VPC = Resource{ID: id}
	}

	// A gateway serves the private subnet it is keyed by and sits in a public one
	// in the same zone. Walked in key order so the choice is stable.
	publicByZone := map[string]string{}

	for _, key := range slices.Sorted(maps.Keys(subnets)) {
		subnet := subnets[key]

		if subnet.Type != contract.SubnetTypePublic.String() || subnet.ID != "" {
			continue
		}

		if _, ok := publicByZone[subnet.Zone]; !ok {
			publicByZone[subnet.Zone] = key
		}
	}

	for _, key := range slices.Sorted(maps.Keys(subnets)) {
		subnet := subnets[key]

		reference, err := contract.NewKey(key)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive subnet %q", key)
		}

		subnetType, err := contract.ParseSubnetType(subnet.Type)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive subnet %q", key)
		}

		// An adopted subnet keeps the operator's own routing.
		if subnet.ID != "" {
			network.Subnets[key] = SubnetResource{ID: subnet.ID, Public: subnetType.IsPublic()}
			continue
		}

		resource := named(Subnet(s, reference, subnetType))

		network.Subnets[key] = SubnetResource{
			Name:   resource.Name,
			Tags:   resource.Tags,
			Public: subnetType.IsPublic(),
		}

		network.RouteTables[key] = named(RouteTable(s, reference))

		if subnetType.IsPublic() {
			// Derived here: the gateway exists only if a public subnet does.
			network.InternetGateway = named(InternetGateway(s))
			continue
		}

		if subnet.Egress != "" {
			network.NATGateways[key] = NATGatewayResource{ID: subnet.Egress}
			continue
		}

		gateway := named(NATGateway(s, reference))
		address := named(ElasticIP(s, reference))

		network.NATGateways[key] = NATGatewayResource{
			Name:    gateway.Name,
			Tags:    gateway.Tags,
			Subnet:  publicByZone[subnet.Zone],
			Address: &address,
		}
	}

	return network, nil
}
