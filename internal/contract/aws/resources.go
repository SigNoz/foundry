package aws

// Resource is one provisioned thing: what to call it and what to stamp on it.
type Resource struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	// Set when the resource is adopted rather than created, in which case the
	// casting stamps nothing on it.
	ID string `json:"id,omitempty"`
}

// SubnetResource resolves a declared subnet.
type SubnetResource struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	ID string `json:"id,omitempty"`

	Public bool `json:"public"`
}

// NATGatewayResource is the egress path of one private subnet.
type NATGatewayResource struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	ID string `json:"id,omitempty"`

	// The public subnet it sits in, in the same zone as the subnet it serves.
	Subnet string `json:"subnet,omitempty"`

	Address *Resource `json:"address,omitempty"`
}

// NetworkResources is the network every substrate shares, embedded by each
// mode beside what only that mode provisions.
type NetworkResources struct {
	VPC Resource `json:"vpc,omitzero"`

	InternetGateway Resource `json:"internetGateway,omitzero"`

	// Keyed by the subnet reference. The maps below are not parallel to this
	// one: a public subnet has no NAT gateway, and neither has an adopted one.
	Subnets map[string]SubnetResource `json:"subnets,omitempty"`

	RouteTables map[string]Resource `json:"routeTables,omitempty"`

	NATGateways map[string]NATGatewayResource `json:"natGateways,omitempty"`
}
