package aws

// Resource is one provisioned thing, resolved: what to call it and what to
// stamp on it.
type Resource struct {
	Name string `json:"name,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	// Set when adopted rather than created; the casting then stamps nothing.
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

	// The public subnet it sits in, in the same zone as the one it serves.
	Subnet string `json:"subnet,omitempty"`

	Address *Resource `json:"address,omitempty"`
}

// NetworkResources is the network every substrate shares. A mode's document
// embeds it beside what only that mode provisions.
type NetworkResources struct {
	VPC Resource `json:"vpc,omitzero"`

	InternetGateway Resource `json:"internetGateway,omitzero"`

	// Keyed by the subnet reference they serve. Not parallel: a public subnet
	// has no NAT gateway, and neither has an adopted one.
	Subnets map[string]SubnetResource `json:"subnets,omitempty"`

	RouteTables map[string]Resource `json:"routeTables,omitempty"`

	NATGateways map[string]NATGatewayResource `json:"natGateways,omitempty"`
}
