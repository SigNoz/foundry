package infrastructure

// ResourceConfig is the resource requirement document (resource.yaml): the
// canonical internal representation of what a substrate shaped for the
// resource kind must provide. It speaks criteria only; platform vocabulary
// never enters it (machines are resolved by castings).
type ResourceConfig struct {
	// Storage the resource requires from the substrate.
	Storage ResourceConfigStorage `json:"storage"`

	// Node groups the resource requires from the substrate.
	NodeGroups []ResourceConfigNodeGroup `json:"nodeGroups" patchStrategy:"merge" patchMergeKey:"name"`
}

// ResourceConfigStorage describes the storage requirement.
type ResourceConfigStorage struct {
	// Whether the resource persists data.
	Persistent *bool `json:"persistent,omitempty"`
}

// ResourceConfigNodeGroup sizes a pool of nodes as criteria, never as
// machine types.
type ResourceConfigNodeGroup struct {
	// Name of the node group.
	Name string `json:"name"`

	// Count of nodes.
	Count *int `json:"count,omitempty"`

	// VCPUs per node.
	VCPUs *int `json:"vcpus,omitempty"`

	// Memory per node in GiB.
	Memory *int `json:"memory,omitempty"`

	// Disk per node in GiB.
	Disk *int `json:"disk,omitempty"`
}
