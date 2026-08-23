package resourcemolding

import (
	"context"
	"log/slog"
	"maps"
	"net/netip"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	infrastructuremolding "github.com/signoz/foundry/internal/molding/infrastructure"
)

// ResourceConfigName is the document a substrate is described by.
const ResourceConfigName = "resource.yaml"

// The groups the baseline declares. A casting keys its contribution to these.
const (
	GroupPersistent = "persistent"
	GroupEphemeral  = "ephemeral"
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

// MoldV1Alpha1 settles the document from the baseline, the casting's
// contribution and the operator's spec, then validates what settled. Names and
// tags are a casting's, derived from this at forge time.
func (molding *resourceMolding) MoldV1Alpha1(ctx context.Context, config *infrastructure.Casting) error {
	status := &config.Spec.Resource.Status

	// Baseline for the resource substrate
	baseline := &infrastructure.ResourceConfig{
		Networking: infrastructure.ResourceConfigNetworking{NetworkCIDR: "10.0.0.0/16"},
		InstanceGroups: map[string]infrastructure.ResourceConfigInstanceGroup{
			GroupPersistent: {
				Storage:    contract.StorageClassPersistent.String(),
				MinSize:    v1alpha1.IntPtr(3),
				MaxSize:    v1alpha1.IntPtr(3),
				RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
				DataVolume: &infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(50)},
			},
			GroupEphemeral: {
				Storage:    contract.StorageClassEphemeral.String(),
				MinSize:    v1alpha1.IntPtr(1),
				MaxSize:    v1alpha1.IntPtr(1),
				RootVolume: infrastructure.ResourceConfigVolume{Size: v1alpha1.IntPtr(30)},
			},
		},
	}

	baselineDoc, err := domain.MarshalYAML(baseline)
	if err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to marshal resource config")
	}

	doc := string(baselineDoc)

	// Enricher deltas first, keeping casting-specific keys, then the operator's
	// spec, which wins.
	for _, override := range []string{
		status.Config.Data[ResourceConfigName],
		config.Spec.Resource.Spec.Config.Data[ResourceConfigName],
	} {
		if override == "" {
			continue
		}

		doc, err = domain.MergeYAML(doc, override)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to merge resource config override")
		}
	}

	declaration := &infrastructure.ResourceConfig{}
	if err := domain.UnmarshalYAML([]byte(doc), declaration); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to unmarshal resolved resource config")
	}

	// Deriving names from it is the casting's, at forge time; the molding only
	// checks the name can be a substrate at all.
	if _, err := contract.NewSubstrate(config.Metadata.Name); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "failed to resolve the substrate being provisioned")
	}

	if err := validate(declaration); err != nil {
		return err
	}

	if status.Config.Data == nil {
		status.Config.Data = make(map[string]string)
	}
	status.Config.Data[ResourceConfigName] = doc

	return nil
}

// validate checks the shared shape; casting-specific keys pass through.
func validate(declaration *infrastructure.ResourceConfig) error {
	if err := validateNetworking(declaration.Networking); err != nil {
		return err
	}

	return validateInstanceGroups(declaration)
}

func validateNetworking(networking infrastructure.ResourceConfigNetworking) error {
	if networking.NetworkID == "" {
		if _, err := netip.ParsePrefix(networking.NetworkCIDR); err != nil {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config networkCIDR %q is not a CIDR block", networking.NetworkCIDR)
		}
	}

	if len(networking.Subnets) == 0 {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config states no subnets: a substrate cannot place a workload without one, and a zone has no safe default")
	}

	// A NAT gateway sits in a public subnet in its own zone.
	publicZones := map[string]struct{}{}
	private := 0

	for _, key := range slices.Sorted(maps.Keys(networking.Subnets)) {
		subnet := networking.Subnets[key]

		if _, err := contract.NewKey(key); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "resource config subnet %q is not a usable reference", key)
		}

		subnetType, err := contract.ParseSubnetType(subnet.Type)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "resource config subnet %q states no usable type", key)
		}

		if subnet.Zone == "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q states no zone", key)
		}

		// A network is adopted whole. Half of one leaves foundry routing subnets
		// it did not create.
		if networking.NetworkID != "" && subnet.ID == "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config adopts network %q, so subnet %q states its own id", networking.NetworkID, key)
		}

		if networking.NetworkID == "" && subnet.ID != "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q states an id, but the network it belongs to is created by foundry", key)
		}

		if subnet.ID == "" {
			if _, err := netip.ParsePrefix(subnet.CIDR); err != nil {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q has cidr %q, which is not a CIDR block", key, subnet.CIDR)
			}
		} else if subnet.Egress != "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q is adopted, so its egress is not foundry's to state", key)
		}

		if subnetType.IsPublic() {
			if subnet.Egress != "" {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q is public, so it states no egress", key)
			}

			if subnet.ID == "" {
				publicZones[subnet.Zone] = struct{}{}
			}

			continue
		}

		private++
	}

	if private == 0 {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config states no private subnet: workloads are never placed in a public one")
	}

	for _, key := range slices.Sorted(maps.Keys(networking.Subnets)) {
		subnet := networking.Subnets[key]

		// An adopted subnet carries its own routing.
		if subnet.Type == contract.SubnetTypePublic.String() || subnet.ID != "" || subnet.Egress != "" {
			continue
		}

		if _, ok := publicZones[subnet.Zone]; !ok {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config subnet %q needs egress but zone %q has no public subnet to place a gateway in", key, subnet.Zone)
		}
	}

	return nil
}

func validateInstanceGroups(declaration *infrastructure.ResourceConfig) error {
	if len(declaration.InstanceGroups) == 0 {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config states no instance groups")
	}

	for _, key := range slices.Sorted(maps.Keys(declaration.InstanceGroups)) {
		group := declaration.InstanceGroups[key]

		if _, err := contract.NewKey(key); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "resource config instance group %q is not a usable reference", key)
		}

		storage, err := contract.ParseStorageClass(group.Storage)
		if err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInvalidInput, "resource config instance group %q states no usable storage class", key)
		}

		if group.MachineType == "" {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q states no machineType", key)
		}

		if group.MinSize == nil || group.MaxSize == nil || group.RootVolume.Size == nil {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q is incomplete", key)
		}

		if *group.MaxSize < *group.MinSize {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q has maxSize below minSize", key)
		}

		if storage.IsPinned() && *group.MinSize != *group.MaxSize {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q is pinned, so minSize and maxSize must be equal", key)
		}

		if storage.RequiresDataVolume() {
			if group.DataVolume == nil || group.DataVolume.Size == nil {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q must state a dataVolume size", key)
			}
		} else if group.DataVolume != nil {
			return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q cannot state a dataVolume", key)
		}

		for _, reference := range group.Subnets {
			subnet, ok := declaration.Networking.Subnets[reference]

			if !ok {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q is placed in subnet %q, which is not declared", key, reference)
			}

			if subnet.Type == contract.SubnetTypePublic.String() {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "resource config instance group %q is placed in subnet %q, which is public", key, reference)
			}
		}
	}

	return nil
}
