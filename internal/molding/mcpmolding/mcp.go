package mcpmolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.Molding = (*mcp)(nil)

type mcp struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *mcp {
	return &mcp{
		logger: logger,
	}
}

func (molding *mcp) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindMCP
}

func (molding *mcp) MoldV1Alpha1(ctx context.Context, config *installation.Casting) error {
	if !config.Spec.MCP.Spec.IsEnabled() {
		return nil
	}

	if config.Spec.MCP.Status.Env == nil {
		config.Spec.MCP.Status.Env = make(map[string]string)
	}

	// Run as a long-lived HTTP server on port 8000 (rather than the default stdio
	// transport, which is spawned per client).
	config.Spec.MCP.Status.Env["TRANSPORT_MODE"] = "http"
	config.Spec.MCP.Status.Env["MCP_SERVER_PORT"] = "8000"

	// Point the server at the co-located SigNoz apiserver. The address is enriched
	// as a tcp:// address; the MCP server wants an http:// URL. The SigNoz API key is
	// deliberately NOT set here: in http mode it is supplied per request by the MCP
	// client as a SIGNOZ-API-KEY header, so no secret lives in the deployment.
	if len(config.Spec.Signoz.Status.Addresses.APIServer) > 0 {
		addrs, err := domain.ParseAddresses(config.Spec.Signoz.Status.Addresses.APIServer)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to parse signoz apiserver addresses")
		}

		if val, ok := config.Spec.MCP.Spec.Env["SIGNOZ_URL"]; ok {
			molding.logger.WarnContext(ctx, "SIGNOZ_URL is going to be overridden", slog.String("value", val))
		}
		config.Spec.MCP.Status.Env["SIGNOZ_URL"] = domain.MustNewAddress("http", addrs[0].Host(), addrs[0].Port()).String()
	}

	return nil
}
