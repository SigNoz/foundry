package installation

import "github.com/signoz/foundry/api/v1alpha1"

type MCP struct {
	// Specification for the MCP server.
	Spec v1alpha1.MoldingSpec `json:"spec" yaml:"spec" description:"Specification for the MCP server"`

	// Status of the MCP server.
	Status MCPStatus `json:"status" yaml:"status,omitempty" description:"Status of the MCP server"`

	_ struct{} `additionalProperties:"false"`
}

type MCPStatus struct {
	v1alpha1.MoldingStatus `json:",inline" yaml:",inline"`

	Addresses MCPStatusAddresses `json:"addresses" yaml:"addresses,omitempty" description:"Addresses of the MCP server"`

	_ struct{} `additionalProperties:"false"`
}

type MCPStatusAddresses struct {
	// HTTP addresses.
	HTTP []string `json:"http" yaml:"http" description:"HTTP addresses"`

	_ struct{} `additionalProperties:"false"`
}

func DefaultMCP() MCP {
	return MCP{
		Spec: v1alpha1.MoldingSpec{
			Enabled: v1alpha1.BoolPtr(false),
			Cluster: v1alpha1.TypeCluster{
				Replicas: v1alpha1.IntPtr(1),
			},
			Version: "latest",
			Image:   "signoz/signoz-mcp-server:latest",
		},
	}
}
