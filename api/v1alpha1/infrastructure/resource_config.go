package infrastructure

// ResourceConfig is the resource requirement document, written as resource.yaml:
// what a substrate shaped for the resource kind must provide.
type ResourceConfig struct {
	// NodeGroups is keyed by storage class, a group's only identity: a consuming
	// casting selects nodes by class and cannot name a group, so a second group
	// of the same class would be unreachable.
	NodeGroups map[StorageClass]ResourceConfigNodeGroup `json:"nodeGroups" description:"Node groups the resource requires from the substrate, keyed by storage class"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigNodeGroup sizes a pool of nodes in the vocabulary every node-pool
// abstraction already uses -- kOps', GKE's and eksctl's terms -- narrowed to what
// foundry has to understand. Capacity is criteria so the document stays portable,
// with machineType as the escape hatch when a concrete type is wanted.
type ResourceConfigNodeGroup struct {
	// MinSize and MaxSize are equal on a pinned group.
	MinSize *int `json:"minSize,omitempty" minimum:"0" description:"Minimum number of nodes in the group"`

	MaxSize *int `json:"maxSize,omitempty" minimum:"0" description:"Maximum number of nodes in the group"`

	MachineType string `json:"machineType,omitempty" description:"Provider machine type; empty resolves one from cpu and memory" example:"m5.large"`

	CPU *int `json:"cpu,omitempty" minimum:"1" description:"CPUs per node, used when machineType is not stated"`

	Memory *int `json:"memory,omitempty" minimum:"1" description:"Memory per node in GB, used when machineType is not stated"`

	RootVolume ResourceConfigVolume `json:"rootVolume,omitzero" description:"The disk each node boots from"`

	DataVolume *ResourceConfigVolume `json:"dataVolume,omitempty" description:"Volume attached to each node that outlives it; persistent storage class only"`

	_ struct{} `additionalProperties:"false"`
}

type ResourceConfigVolume struct {
	Size *int `json:"size,omitempty" minimum:"1" description:"Size of the volume in GB"`

	_ struct{} `additionalProperties:"false"`
}
