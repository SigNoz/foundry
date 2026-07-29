package infrastructure

// ResourceConfig is the resource requirement document (resource.yaml): the
// canonical internal representation of what a substrate shaped for the
// resource kind must provide. It speaks criteria only; platform vocabulary
// never enters it (machines are resolved by the platform).
type ResourceConfig struct {
	// Node groups the resource requires from the substrate.
	NodeGroups []ResourceConfigNodeGroup `json:"nodeGroups" patchStrategy:"merge" patchMergeKey:"name" description:"Node groups the resource requires from the substrate"`

	_ struct{} `additionalProperties:"false"`
}

// ResourceConfigNodeGroup sizes a pool of nodes as criteria, never as
// machine types.
type ResourceConfigNodeGroup struct {
	// Name of the node group.
	Name string `json:"name" description:"Name of the node group"`

	// Whether this group's nodes persist data.
	Persistent *bool `json:"persistent,omitempty" description:"Whether this group's nodes persist data"`

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
