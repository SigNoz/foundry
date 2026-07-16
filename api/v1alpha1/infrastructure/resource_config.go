package infrastructure

// ResourceConfig is the resource requirement document (resource.yaml): the
// canonical internal representation of what a substrate shaped for the
// resource kind must provide. It speaks criteria only; platform vocabulary
// never enters it (machines are resolved by the platform).
type ResourceConfig struct {
	// Storage the resource requires from the substrate.
	Storage ResourceConfigStorage `json:"storage" description:"Storage the resource requires from the substrate"`

	// Node groups the resource requires from the substrate.
	NodeGroups []ResourceConfigNodeGroup `json:"nodeGroups" patchStrategy:"merge" patchMergeKey:"name" description:"Node groups the resource requires from the substrate"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigStorage describes the storage requirement.
type ResourceConfigStorage struct {
	// Whether the resource persists data.
	Persistent *bool `json:"persistent,omitempty" description:"Whether the resource persists data"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigNodeGroup sizes a pool of nodes as criteria, never as
// machine types.
type ResourceConfigNodeGroup struct {
	// Name of the node group.
	Name string `json:"name" description:"Name of the node group"`

	// Count of nodes.
	Count *int `json:"count,omitempty" description:"Count of nodes"`

	// VCPUs per node.
	VCPUs *int `json:"vcpus,omitempty" description:"VCPUs per node"`

	// Memory per node in GiB.
	Memory *int `json:"memory,omitempty" description:"Memory per node in GiB"`

	// Disk per node in GiB.
	Disk *int `json:"disk,omitempty" description:"Disk per node in GiB"`

	_ struct{} `additionalProperties:"false"`
}
