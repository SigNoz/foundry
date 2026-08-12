package aws

import (
	"maps"
	"slices"

	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/internal/contract"
	"github.com/signoz/foundry/internal/errors"
)

// PlacedGroup is one declared instance group with its placement resolved: the
// subnets its nodes go in, stated on the declaration or falling back to every
// private one.
type PlacedGroup struct {
	Key      string
	Declared infrastructure.ResourceConfigInstanceGroup
	Storage  contract.StorageClass
	Group    contract.NodeGroup
	Subnets  []string
}

// PlaceInstanceGroups resolves every declared group in key order. Every
// substrate places groups the same way; what sits on top of a placed group is
// the substrate's own.
func PlaceInstanceGroups(declaration *infrastructure.ResourceConfig) ([]PlacedGroup, error) {
	// A group that names no subnet is placed across every private one.
	placement := []string{}

	for _, key := range slices.Sorted(maps.Keys(declaration.Networking.Subnets)) {
		if declaration.Networking.Subnets[key].Type != contract.SubnetTypePublic.String() {
			placement = append(placement, key)
		}
	}

	placed := make([]PlacedGroup, 0, len(declaration.InstanceGroups))

	for _, key := range slices.Sorted(maps.Keys(declaration.InstanceGroups)) {
		declared := declaration.InstanceGroups[key]

		reference, err := contract.NewKey(key)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive instance group %q", key)
		}

		storage, err := contract.ParseStorageClass(declared.Storage)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to derive instance group %q", key)
		}

		subnets := declared.Subnets

		if len(subnets) == 0 {
			subnets = placement
		}

		if len(subnets) == 0 {
			return nil, errors.Newf(errors.TypeInvalidInput, "failed to derive instance group %q: there is no private subnet to place it in", key)
		}

		placed = append(placed, PlacedGroup{
			Key:      key,
			Declared: declared,
			Storage:  storage,
			Group:    contract.NewNodeGroup(reference, storage),
			Subnets:  subnets,
		})
	}

	return placed, nil
}
