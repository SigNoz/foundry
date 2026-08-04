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

// Node groups merge by name so a contribution or an operator override states
// only the group and the fields it changes, instead of restating every group
// to avoid deleting the ones it left out.
var resourceConfigListTypes = domain.ListTypes{"nodeGroups": domain.ListTypeMap("name")}

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
			NodeGroups: []infrastructure.ResourceConfigNodeGroup{
				// Three persistent nodes cover the default installation's
				// stateful set: one keeper, the metadata node, one store node.
				// A group holding stateful identities is pinned, so its bounds
				// are equal -- there is nothing to autoscale when every node
				// owns a claimed volume.
				{
					Name:       infrastructure.StorageClassPersistent.String(),
					Storage:    infrastructure.StorageClassPersistent,
					MinSize:    v1alpha1.IntPtr(3),
					MaxSize:    v1alpha1.IntPtr(3),
					CPU:        v1alpha1.IntPtr(2),
					Memory:     v1alpha1.IntPtr(8),
					RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
					DataVolume: &infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(50)},
				},
				{
					Name:       infrastructure.StorageClassEphemeral.String(),
					Storage:    infrastructure.StorageClassEphemeral,
					MinSize:    v1alpha1.IntPtr(1),
					MaxSize:    v1alpha1.IntPtr(1),
					CPU:        v1alpha1.IntPtr(2),
					Memory:     v1alpha1.IntPtr(4),
					RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
				},
			},
		}
	case infrastructure.ResourceKindCollectionAgent:
		status.Addresses.OTLP = append([]string{otlpGRPCAddress, otlpHTTPAddress}, status.Addresses.OTLP...)
		baseline = &infrastructure.ResourceConfig{
			NodeGroups: []infrastructure.ResourceConfigNodeGroup{
				{
					Name:       infrastructure.StorageClassEphemeral.String(),
					Storage:    infrastructure.StorageClassEphemeral,
					MinSize:    v1alpha1.IntPtr(1),
					MaxSize:    v1alpha1.IntPtr(1),
					CPU:        v1alpha1.IntPtr(2),
					Memory:     v1alpha1.IntPtr(4),
					RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
				},
			},
		}
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported resource kind %q", config.Spec.Resource.Kind)
	}

	baselineDoc, err := domain.MarshalYAML(baseline)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to marshal resource config")
	}

	doc := string(baselineDoc)

	// Contributions (enricher deltas) merge first so casting-specific keys
	// survive, then the operator's own spec, which wins: spec beats status
	// wherever they disagree.
	for _, override := range []string{
		status.Config.Data[ResourceConfigName],
		config.Spec.Resource.Spec.Config.Data[ResourceConfigName],
	} {
		if override == "" {
			continue
		}

		doc, err = domain.StrategicMergeYAML(doc, override, resourceConfigListTypes)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to merge resource config override")
		}
	}

	if err := validate(doc); err != nil {
		return err
	}

	if status.Config.Data == nil {
		status.Config.Data = make(map[string]string)
	}
	status.Config.Data[ResourceConfigName] = doc

	return nil
}

// validate checks the shared shape of the resolved document; casting-specific
// keys pass through unchecked.
func validate(doc string) error {
	config := &infrastructure.ResourceConfig{}
	if err := domain.UnmarshalYAML([]byte(doc), config); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to unmarshal resolved resource config")
	}

	for _, group := range config.NodeGroups {
		if group.MinSize == nil || group.MaxSize == nil || group.RootVolume.Size == nil {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q in resource config is incomplete", group.Name)
		}

		if *group.MaxSize < *group.MinSize {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q has maxSize below minSize", group.Name)
		}

		// A machine is named outright or resolved from criteria; one of the
		// two has to be stated or there is nothing to launch.
		if group.MachineType == "" && (group.CPU == nil || group.Memory == nil) {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q states neither machineType nor cpu and memory", group.Name)
		}

		// An unknown class cannot reach here: it fails at unmarshal. What is
		// left is a group that named none at all.
		if group.Storage.String() == "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q states no storage class", group.Name)
		}

		if group.Storage.RequiresDataVolume() {
			if group.DataVolume == nil || group.DataVolume.Size == nil {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q is %s, so it must state a dataVolume size", group.Name, group.Storage)
			}
		} else if group.DataVolume != nil {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q is %s, so it cannot state a dataVolume", group.Name, group.Storage)
		}

		if group.Storage.IsPinned() && *group.MinSize != *group.MaxSize {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "node group %q is %s, so minSize and maxSize must be equal", group.Name, group.Storage)
		}
	}

	return nil
}
