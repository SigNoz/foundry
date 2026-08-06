package infrastructure

import "github.com/signoz/foundry/api/v1alpha1"

// ResourceConfig is the requirement document, written as resource.yaml.
//
// Everything above Resources is declared: a molding baseline, a casting's
// contribution, then the operator's spec, which wins. Resources is derived once
// that settles.
type ResourceConfig struct {
	Networking ResourceConfigNetworking `json:"networking,omitzero" description:"The network the substrate runs in"`

	IAM ResourceConfigIAM `json:"iam,omitzero" description:"Identity the substrate's workloads assume"`

	CloudLabels map[string]string `json:"cloudLabels,omitempty" description:"Tags applied to every resource the substrate provisions"`

	InstanceGroups map[string]ResourceConfigInstanceGroup `json:"instanceGroups,omitempty" description:"Pools of nodes the resource requires, keyed by a reference of your choosing"`

	// Stating this is an error, not an override: a name that disagrees with the
	// tag derived beside it matches nothing.
	Resources *ResourceConfigResources `json:"resources,omitempty" description:"Derived names and tags for everything the substrate provisions; written by foundry"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigNetworking follows kOps' NetworkingSpec.
type ResourceConfigNetworking struct {
	// A network is adopted whole: every subnet then states its own id.
	NetworkID string `json:"networkID,omitempty" description:"Provider ID of an existing network to adopt; empty creates one" example:"vpc-0a1b2c3d"`

	NetworkCIDR string `json:"networkCIDR,omitempty" description:"CIDR block for the network" example:"10.0.0.0/16"`

	Subnets map[string]ResourceConfigSubnet `json:"subnets,omitempty" description:"Subnets carved out of the network, keyed by a reference of your choosing"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigSubnet follows kOps' ClusterSubnetSpec. Zone has no default:
// letters are not contiguous within a region and the mapping is per-account.
type ResourceConfigSubnet struct {
	Type v1alpha1.SubnetType `json:"type,omitzero" description:"Whether the subnet routes to an internet gateway"`

	Zone string `json:"zone,omitempty" description:"Availability zone the subnet lives in" example:"us-east-1a"`

	CIDR string `json:"cidr,omitempty" description:"CIDR block for the subnet, carved out of the network" example:"10.0.0.0/19"`

	// Private subnets only; empty creates a gateway in a public subnet of the
	// same zone.
	Egress string `json:"egress,omitempty" description:"Provider ID of an existing NAT gateway this private subnet routes through; empty creates one" example:"nat-0a1b2c3d"`

	ID string `json:"id,omitempty" description:"Provider ID of an existing subnet to adopt; empty creates one" example:"subnet-0a1b2c3d"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigIAM constrains the roles the substrate creates; which roles
// exist is the platform's, and their names are derived.
type ResourceConfigIAM struct {
	PermissionsBoundary string `json:"permissionsBoundary,omitempty" description:"Policy ARN attached as the permissions boundary of every role the substrate creates"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigInstanceGroup follows kOps' InstanceGroupSpec, narrowed to what
// foundry has to understand.
type ResourceConfigInstanceGroup struct {
	Storage v1alpha1.StorageClass `json:"storage,omitzero" description:"Durability of the group's storage, and the only fact about it a consuming casting can select on"`

	MachineType string `json:"machineType,omitempty" description:"Provider machine type for each node in the group" example:"m5.large"`

	// MinSize and MaxSize are equal on a pinned group.
	MinSize *int `json:"minSize,omitempty" minimum:"0" description:"Minimum number of nodes in the group"`

	MaxSize *int `json:"maxSize,omitempty" minimum:"0" description:"Maximum number of nodes in the group"`

	// References into networking.subnets; a pinned group's nodes are laid out
	// across them in order.
	Subnets []string `json:"subnets,omitempty" description:"Subnet references the group's nodes are placed in"`

	RootVolume ResourceConfigVolume `json:"rootVolume,omitzero" description:"The disk each node boots from, which dies with it"`

	DataVolume *ResourceConfigVolume `json:"dataVolume,omitempty" description:"Volume attached to each node that outlives it; persistent storage class only"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigVolume follows kOps' VolumeSpec.
type ResourceConfigVolume struct {
	Size *int `json:"size,omitempty" minimum:"1" description:"Size of the volume in GB"`

	Type string `json:"type,omitempty" description:"Provider volume type" example:"gp3"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResources is every name and tag derived from the declaration.
// Templates interpolate these rather than assembling their own.
type ResourceConfigResources struct {
	Cluster ResourceConfigResource `json:"cluster,omitzero"`

	VPC ResourceConfigResource `json:"vpc,omitzero"`

	InternetGateway ResourceConfigResource `json:"internetGateway,omitzero"`

	SecurityGroup ResourceConfigResource `json:"securityGroup,omitzero"`

	SecurityGroupRules map[string]ResourceConfigResource `json:"securityGroupRules,omitempty"`

	Roles map[string]ResourceConfigResource `json:"roles,omitempty"`

	InstanceProfile ResourceConfigResource `json:"instanceProfile,omitzero"`

	// Keyed by the subnet reference they serve. Not parallel: a public subnet
	// has no NAT gateway, and neither has an adopted one.
	Subnets map[string]ResourceConfigResourceSubnet `json:"subnets,omitempty"`

	RouteTables map[string]ResourceConfigResource `json:"routeTables,omitempty"`

	NATGateways map[string]ResourceConfigResourceNATGateway `json:"natGateways,omitempty"`

	InstanceGroups map[string]ResourceConfigResourceGroup `json:"instanceGroups,omitempty"`

	// Stamped after provisioning by whatever claims a resource; reconciling
	// them reverts a live claim on every apply.
	IgnoredTags []string `json:"ignoredTags,omitempty"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResource is one derived thing: what to call it and what to
// stamp on it.
type ResourceConfigResource struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	// Set when adopted rather than created; the casting then stamps nothing.
	ID string `json:"id,omitempty"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResourceSubnet resolves a declared subnet.
type ResourceConfigResourceSubnet struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	ID string `json:"id,omitempty"`

	Public bool `json:"public"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResourceNATGateway is the egress path of one private subnet.
type ResourceConfigResourceNATGateway struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	ID string `json:"id,omitempty"`

	// The public subnet it sits in, in the same zone as the one it serves.
	Subnet string `json:"subnet,omitempty"`

	Address *ResourceConfigResource `json:"address,omitempty"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResourceGroup is what a declared instance group resolves to. A
// pinned group has Nodes and no autoscaling group; a scaling one is the
// reverse, which is how a casting tells them apart.
type ResourceConfigResourceGroup struct {
	Storage v1alpha1.StorageClass `json:"storage,omitzero"`

	// The tag match that finds this group's nodes. The substrate advertises it
	// wherever the platform does placement.
	Selector map[string]string `json:"selector,omitempty"`

	Subnets []string `json:"subnets,omitempty"`

	LaunchTemplate *ResourceConfigResource `json:"launchTemplate,omitempty"`

	AutoscalingGroup *ResourceConfigResource `json:"autoscalingGroup,omitempty"`

	Nodes []ResourceConfigResourceNode `json:"nodes,omitempty"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigResourceNode is one node of a pinned group. Its volume is
// stated inside it so the two cannot land in different zones.
type ResourceConfigResourceNode struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	Ordinal int `json:"ordinal"`

	Subnet string `json:"subnet,omitempty"`

	Volume *ResourceConfigResource `json:"volume,omitempty"`

	_ struct{} `additionalProperties:"false"`
}
