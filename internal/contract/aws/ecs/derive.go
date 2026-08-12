// Package ecs derives the resource set for a substrate that owns its instances.
// A pinned group resolves to named nodes with named volumes, an elastic group
// to a pool the substrate scales.
package ecs

import (
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/contract/aws"
)

// What the substrate's own rules exist for. Each is a name segment.
var (
	purposeIntraCluster = contract.MustNewKey("intra-cluster")
	purposeAllOutbound  = contract.MustNewKey("all-outbound")
)

// Resources is the settled declaration beside every name and tag derived from
// it. Templates interpolate it and assemble no name or tag of their own.
type Resources struct {
	Declaration *infrastructure.ResourceConfig

	Cluster aws.Resource

	Roles map[string]aws.Resource

	Network *aws.NetworkResources

	SecurityGroup aws.Resource

	SecurityGroupRules map[string]aws.Resource

	InstanceProfile aws.Resource

	// A declared group resolves into one of these two: Pinned when the substrate
	// owns each instance and its volume, Pools when it owns the pool alone.
	Pinned map[string]PinnedGroup

	Pools map[string]PoolGroup

	IgnoredTags []string
}

type PinnedGroup struct {
	Declared infrastructure.ResourceConfigInstanceGroup

	Storage contract.StorageClass

	Selector map[string]string

	Nodes []Node
}

// Node is one node of a pinned group. Its volume is stated inside it, so the
// two cannot land in different zones.
type Node struct {
	Name string

	Ordinal int

	Subnet string

	Tags map[string]string

	Volume aws.Resource
}

type PoolGroup struct {
	Declared infrastructure.ResourceConfigInstanceGroup

	Storage contract.StorageClass

	Selector map[string]string

	Subnets []string

	LaunchTemplate aws.Resource

	AutoscalingGroup aws.Resource
}

func Derive(s contract.Substrate, declaration *infrastructure.ResourceConfig, labels map[string]string) (*Resources, error) {
	named := aws.Stamper(declaration, labels)

	resources := &Resources{
		Declaration:     declaration,
		Cluster:         named(aws.Cluster(s)),
		SecurityGroup:   named(aws.SecurityGroup(s, aws.RoleTask)),
		InstanceProfile: named(aws.InstanceProfile(s, aws.RoleNode)),
		SecurityGroupRules: map[string]aws.Resource{
			purposeIntraCluster.String(): named(aws.SecurityGroupRule(s, aws.RoleTask, purposeIntraCluster)),
			purposeAllOutbound.String():  named(aws.SecurityGroupRule(s, aws.RoleTask, purposeAllOutbound)),
		},
		Roles: map[string]aws.Resource{},

		// The claim tag is stamped after provisioning, so reconciling it would
		// revert a live claim on every apply.
		IgnoredTags: []string{aws.Tag(contract.TagKeyIdentities)},
	}

	// The node's own credential only, without which the agent cannot register
	// the instance with the cluster. Workload identity belongs to the workload.
	resources.Roles[aws.RoleNode.String()] = named(aws.IAMRole(s, aws.RoleNode))

	network, err := aws.Networking(s, named, declaration)
	if err != nil {
		return nil, err
	}

	resources.Network = network

	placed, err := aws.PlaceInstanceGroups(declaration)
	if err != nil {
		return nil, err
	}

	resources.Pinned = map[string]PinnedGroup{}
	resources.Pools = map[string]PoolGroup{}

	for _, placement := range placed {
		selector := aws.Filter(s.Select().WithStorage(placement.Storage))

		if !placement.Storage.IsPinned() {
			resources.Pools[placement.Key] = PoolGroup{
				Declared:         placement.Declared,
				Storage:          placement.Storage,
				Selector:         selector,
				Subnets:          placement.Subnets,
				LaunchTemplate:   named(aws.LaunchTemplate(s, placement.Group)),
				AutoscalingGroup: named(aws.AutoscalingGroup(s, placement.Group)),
			}

			continue
		}

		group := PinnedGroup{Declared: placement.Declared, Storage: placement.Storage, Selector: selector}

		// Each node carries its volume. The two cannot land in different zones.
		nodes := 0

		if placement.Declared.MinSize != nil {
			nodes = *placement.Declared.MinSize
		}

		for ordinal := range nodes {
			node := named(aws.Node(s, placement.Group, ordinal))

			group.Nodes = append(group.Nodes, Node{
				Name:    node.Name,
				Ordinal: ordinal,
				Subnet:  placement.Subnets[ordinal%len(placement.Subnets)],
				Tags:    node.Tags,
				Volume:  named(aws.Volume(s, placement.Group, ordinal)),
			})
		}

		resources.Pinned[placement.Key] = group
	}

	return resources, nil
}
