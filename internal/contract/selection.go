package contract

// Selection is a substrate narrowed by the facts a consumer can predict: the
// subnet type, the storage class, and the identities claiming a resource.
type Selection struct {
	substrate  Substrate
	subnetType SubnetType
	storage    StorageClass
	identities Identities
}

func (s Substrate) Select() Selection {
	return Selection{substrate: s}
}

func (selection Selection) WithSubnetType(subnetType SubnetType) Selection {
	selection.subnetType = subnetType

	return selection
}

func (selection Selection) WithStorage(storage StorageClass) Selection {
	selection.storage = storage

	return selection
}

func (selection Selection) WithClaims(identities Identities) Selection {
	selection.identities = identities

	return selection
}

// Match is the tag set that finds exactly what the selection narrows to.
func (selection Selection) Match() map[TagKey]string {
	tags := map[TagKey]string{
		TagKeyName: selection.substrate.name,
	}

	if selection.subnetType != (SubnetType{}) {
		tags[TagKeySubnetType] = selection.subnetType.String()
	}

	if selection.storage != (StorageClass{}) {
		tags[TagKeyStorage] = selection.storage.String()
	}

	if len(selection.identities) > 0 {
		tags[TagKeyIdentities] = selection.identities.String()
	}

	return tags
}
