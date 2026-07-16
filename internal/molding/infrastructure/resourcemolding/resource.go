package resourcemolding

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

// The edge conventions of the SigNoz resource kinds.
var (
	otlpGRPCAddress  = domain.MustNewAddress("tcp", "0.0.0.0", 4317).String()
	otlpHTTPAddress  = domain.MustNewAddress("tcp", "0.0.0.0", 4318).String()
	apiServerAddress = domain.MustNewAddress("tcp", "0.0.0.0", 8080).String()
)

// ResourceConfigName is the config document carrying the requirements a
// substrate shaped for the resource kind must satisfy beyond its edge.
const ResourceConfigName = "resource.yaml"

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

// MoldV1Alpha1 writes the kind-level requirement set into the resource
// status: baseline first, preserving entries an enricher has already
// contributed.
func (molding *resourceMolding) MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error {
	status := &config.Spec.Resource.Status

	var baseline *infrastructure.ResourceConfig
	switch config.Spec.Resource.Kind {
	case infrastructure.ResourceKindInstallation:
		status.Addresses.OTLP = append([]string{otlpGRPCAddress, otlpHTTPAddress}, status.Addresses.OTLP...)
		status.Addresses.APIServer = append([]string{apiServerAddress}, status.Addresses.APIServer...)
		baseline = &infrastructure.ResourceConfig{
			Storage: infrastructure.ResourceConfigStorage{Persistent: v1alpha1.BoolPtr(true)},
			NodeGroups: []infrastructure.ResourceConfigNodeGroup{
				{Name: "default", Count: v1alpha1.IntPtr(2), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(8), Disk: v1alpha1.IntPtr(50)},
			},
		}
	case infrastructure.ResourceKindCollectionAgent:
		status.Addresses.OTLP = append([]string{otlpGRPCAddress, otlpHTTPAddress}, status.Addresses.OTLP...)
		baseline = &infrastructure.ResourceConfig{
			Storage: infrastructure.ResourceConfigStorage{Persistent: v1alpha1.BoolPtr(false)},
			NodeGroups: []infrastructure.ResourceConfigNodeGroup{
				{Name: "default", Count: v1alpha1.IntPtr(1), VCPUs: v1alpha1.IntPtr(2), Memory: v1alpha1.IntPtr(4), Disk: v1alpha1.IntPtr(20)},
			},
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported resource kind %q", config.Spec.Resource.Kind)
	}

	if overrides := status.Config.Data[ResourceConfigName]; overrides != "" {
		overrideConfig := &infrastructure.ResourceConfig{}
		if err := domain.UnmarshalYAML([]byte(overrides), overrideConfig); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to unmarshal resource config overrides")
		}
		if err := v1alpha1.Merge(baseline, overrideConfig); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to merge resource config overrides")
		}
	}

	doc, err := domain.MarshalYAML(baseline)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to marshal resource config")
	}

	if status.Config.Data == nil {
		status.Config.Data = make(map[string]string)
	}
	status.Config.Data[ResourceConfigName] = string(doc)

	return nil
}
