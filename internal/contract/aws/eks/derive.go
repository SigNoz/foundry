// Package eks derives the resource set for a substrate whose nodes a managed
// control plane owns. The provider scales each pool and replaces nodes within
// it, so the substrate names the pool and never a node or a volume in it.
// Persistent volumes are provisioned by the workload's own platform against the
// storage class its group advertises.
package eks

import (
	"maps"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/contract/aws"
	"github.com/signoz/foundry/internal/errors"
)

// Tags the kubernetes ecosystem reads off a subnet, not facts foundry stamps.
// A load balancer controller picks its subnets by these and looks for no
// foundry-prefixed spelling.
const (
	tagRoleELB         = "kubernetes.io/role/elb"
	tagRoleInternalELB = "kubernetes.io/role/internal-elb"
	tagClusterPrefix   = "kubernetes.io/cluster/"
)

// tagRoleValue is what the controller expects. Only the key carries meaning.
const tagRoleValue = "1"

// minimumZones is how many availability zones a managed control plane places
// its own interfaces across.
const minimumZones = 2

// Resources is the settled declaration beside every name and tag derived from
// it. Templates interpolate it and assemble no name or tag of their own.
type Resources struct {
	Declaration *infrastructure.ResourceConfig

	Cluster aws.Resource

	Roles map[string]aws.Resource

	Network *aws.NetworkResources

	Groups map[string]Group

	IgnoredTags []string
}

// Group is a pool the provider scales and replaces nodes in. No node in it is
// named, and no volume is attached here.
type Group struct {
	Declared infrastructure.ResourceConfigInstanceGroup

	Storage contract.StorageClass

	Selector map[string]string

	Subnets []string

	NodeGroup aws.Resource
}

func Derive(s contract.Substrate, declaration *infrastructure.ResourceConfig, labels map[string]string) (*Resources, error) {
	if err := checkZoneSpread(declaration); err != nil {
		return nil, err
	}

	named := aws.Stamper(declaration, labels)

	resources := &Resources{
		Declaration: declaration,
		Cluster:     named(aws.Cluster(s)),
		Roles:       map[string]aws.Resource{},

		// The control plane stamps this tag on what it discovers, so reconciling
		// it would revert a live cluster's claim on every apply.
		IgnoredTags: []string{tagClusterPrefix + aws.Cluster(s).Name()},
	}

	// The control plane assumes the first to manage the cluster, a node the
	// second to register with it, and the storage driver the third through pod
	// identity. None belongs to a tenant workload.
	resources.Roles[aws.RoleCluster.String()] = named(aws.IAMRole(s, aws.RoleCluster))
	resources.Roles[aws.RoleNode.String()] = named(aws.IAMRole(s, aws.RoleNode))
	resources.Roles[aws.RoleEBSCSI.String()] = named(aws.IAMRole(s, aws.RoleEBSCSI))

	network, err := aws.Networking(s, named, declaration)
	if err != nil {
		return nil, err
	}

	electSubnetsForLoadBalancers(network)
	resources.Network = network

	placed, err := aws.PlaceInstanceGroups(declaration)
	if err != nil {
		return nil, err
	}

	// One pool per declared group. Bounds and machine type stay on the
	// declaration; the pool's name, its node tag match and its placement are
	// derived here.
	resources.Groups = make(map[string]Group, len(placed))

	for _, placement := range placed {
		resources.Groups[placement.Key] = Group{
			Declared:  placement.Declared,
			Storage:   placement.Storage,
			Selector:  aws.Filter(s.Select().WithStorage(placement.Storage)),
			Subnets:   placement.Subnets,
			NodeGroup: named(aws.NodeGroup(s, placement.Group)),
		}
	}

	return resources, nil
}

func checkZoneSpread(declaration *infrastructure.ResourceConfig) error {
	zones := map[string]struct{}{}
	for _, subnet := range declaration.Networking.Subnets {
		zones[subnet.Zone] = struct{}{}
	}

	if len(zones) < minimumZones {
		return errors.Newf(errors.TypeInvalidInput, "failed to derive substrate: a managed control plane places its interfaces across at least %d availability zones, and the subnets declare %d", minimumZones, len(zones))
	}

	return nil
}

// electSubnetsForLoadBalancers marks which subnets a load balancer may be
// placed in: an internet-facing one in a public subnet, an internal one in a
// private subnet. An adopted subnet keeps its owner's tags and is left alone.
func electSubnetsForLoadBalancers(network *aws.NetworkResources) {
	for key, subnet := range network.Subnets {
		if subnet.Tags == nil {
			continue
		}

		role := tagRoleInternalELB
		if subnet.Public {
			role = tagRoleELB
		}

		tags := maps.Clone(subnet.Tags)
		tags[role] = tagRoleValue

		subnet.Tags = tags
		network.Subnets[key] = subnet
	}
}
