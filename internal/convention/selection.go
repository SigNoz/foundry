package convention

import (
	"github.com/signoz/foundry/api/v1alpha1"
)

// Selection is what a consuming casting looks for: a substrate, narrowed by the
// facts it knows.
type Selection struct {
	substrate  Substrate
	subnetType v1alpha1.SubnetType
	storage    v1alpha1.StorageClass
	identities Identities
}

func (s Substrate) Select() Selection {
	return Selection{substrate: s}
}

// WithSubnetType narrows to the subnets a workload may be placed in.
func (selection Selection) WithSubnetType(subnetType v1alpha1.SubnetType) Selection {
	selection.subnetType = subnetType

	return selection
}

func (selection Selection) WithStorage(storage v1alpha1.StorageClass) Selection {
	selection.storage = storage

	return selection
}

// WithClaims narrows to the resource holding these identities. See Identities.
func (selection Selection) WithClaims(identities Identities) Selection {
	selection.identities = identities

	return selection
}

// Match is the only place the facts a consumer depends on are decided. A
// resource stamps these plus provenance.
func (selection Selection) Match() map[TagKey]string {
	tags := map[TagKey]string{
		TagKeyName: selection.substrate.name,
	}

	if selection.subnetType != (v1alpha1.SubnetType{}) {
		tags[TagKeySubnetType] = selection.subnetType.String()
	}

	if selection.storage != (v1alpha1.StorageClass{}) {
		tags[TagKeyStorage] = selection.storage.String()
	}

	if len(selection.identities) > 0 {
		tags[TagKeyIdentities] = selection.identities.String()
	}

	return tags
}
