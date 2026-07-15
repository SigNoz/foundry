package resourcemolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

var _ infrastructuremolding.Molding = (*resourceMolding)(nil)

type resourceMolding struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *resourceMolding {
	return &resourceMolding{logger: logger}
}

func (molding *resourceMolding) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindResource
}

// MoldV1Alpha1 molds the unit to be hosted: the exposure and facts every
// casting consumes, dispatched on the resource kind.
func (molding *resourceMolding) MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error {
	status := &config.Spec.Resource.Status

	switch config.Spec.Resource.Kind {
	case infrastructure.ResourceKindInstallation:
		status.Addresses = infrastructure.ResourceStatusAddresses{
			OTLP: []string{":4317", ":4318"},
			UI:   []string{":8080"},
		}
	case infrastructure.ResourceKindCollectionAgent:
		status.Addresses = infrastructure.ResourceStatusAddresses{
			OTLP: []string{":4317", ":4318"},
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported resource kind %q", config.Spec.Resource.Kind)
	}

	return nil
}
