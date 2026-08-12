package infrastructure

// ResourceConfig is the requirement document, written as resource.yaml: a
// molding baseline, a casting's contribution, then the operator's spec, which
// wins. Names and tags are not part of it; a casting derives them from the
// settled declaration at forge time.
type ResourceConfig struct {
	Networking ResourceConfigNetworking `json:"networking,omitzero" description:"The network the substrate runs in"`

	IAM ResourceConfigIAM `json:"iam,omitzero" description:"Identity the substrate's workloads assume"`

	CloudLabels map[string]string `json:"cloudLabels,omitempty" description:"Tags applied to every resource the substrate provisions"`

	InstanceGroups map[string]ResourceConfigInstanceGroup `json:"instanceGroups,omitempty" description:"Pools of nodes the resource requires, keyed by a reference of your choosing"`

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
	Type string `json:"type,omitempty" description:"Whether the subnet routes to an internet gateway: private or public"`

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
	Storage string `json:"storage,omitempty" description:"Durability of the group's storage, persistent or ephemeral, and the only fact about it a consuming casting can select on"`

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
