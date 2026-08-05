package infrastructure

// ResourceConfig is the resource requirement document (resource.yaml): the
// canonical internal representation of what a substrate shaped for the
// resource kind must provide.
type ResourceConfig struct {
	// NodeGroups is keyed by storage class, which is the only identity a group
	// has: a consuming casting selects nodes by class and has no way to name a
	// group, so a second group of the same class would be unreachable.
	NodeGroups map[StorageClass]ResourceConfigNodeGroup `json:"nodeGroups" description:"Node groups the resource requires from the substrate, keyed by storage class"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigNodeGroup sizes a pool of nodes. The vocabulary is the one
// every node-pool abstraction already uses -- machineType, minSize, maxSize
// and per-volume sizes are kOps', GKE's and eksctl's terms -- narrowed to what
// foundry has to understand. Capacity may be stated as criteria (cpu, memory)
// so the document stays portable across providers, with machineType as the
// escape hatch when a concrete type is wanted.
type ResourceConfigNodeGroup struct {
	// MinSize is the smallest the group may be. A pinned group states the
	// same value for both bounds.
	MinSize *int `json:"minSize,omitempty" minimum:"0" description:"Minimum number of nodes in the group"`

	// MaxSize is the largest the group may grow to.
	MaxSize *int `json:"maxSize,omitempty" minimum:"0" description:"Maximum number of nodes in the group"`

	// MachineType names the provider's machine type outright; empty resolves
	// one from cpu and memory against the provider's catalog.
	MachineType string `json:"machineType,omitempty" description:"Provider machine type; empty resolves one from cpu and memory" example:"m5.large"`

	// CPU per node.
	CPU *int `json:"cpu,omitempty" minimum:"1" description:"CPUs per node, used when machineType is not stated"`

	// Memory per node in GB.
	Memory *int `json:"memory,omitempty" minimum:"1" description:"Memory per node in GB, used when machineType is not stated"`

	// RootVolume is the disk each node boots from.
	RootVolume ResourceConfigVolume `json:"rootVolume,omitzero" description:"The disk each node boots from"`

	// DataVolume outlives the node it is attached to. Absent on a group whose
	// nodes keep nothing.
	DataVolume *ResourceConfigVolume `json:"dataVolume,omitempty" description:"Volume attached to each node that outlives it; persistent storage class only"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigVolume sizes one volume.
type ResourceConfigVolume struct {
	// Size of the volume in GB.
	Size *int `json:"size,omitempty" minimum:"1" description:"Size of the volume in GB"`

	_ struct{} `additionalProperties:"false"`
}
