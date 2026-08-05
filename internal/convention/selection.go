package convention

import (
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
)

// Selection is what a consuming casting looks for: a substrate, narrowed by the
// facts it knows. A consumer knows a class rather than an instance, and cannot
// know the group names the producer chose, so no name or resource type appears
// here -- neither reaches a tag.
type Selection struct {
	substrate  Substrate
	storage    infrastructure.StorageClass
	identities Identities
}

func (s Substrate) Select() Selection {
	return Selection{substrate: s}
}

func (selection Selection) WithStorage(storage infrastructure.StorageClass) Selection {
	selection.storage = storage

	return selection
}

// WithClaims narrows to the resource holding these identities. See Identities:
// only a platform with no stateful identity primitive of its own needs this.
func (selection Selection) WithClaims(identities Identities) Selection {
	selection.identities = identities

	return selection
}

// match is the only place the tags a consumer depends on are decided. Resource
// stamps these plus provenance, so the two cannot disagree.
func (selection Selection) match() Tags {
	tags := Tags{
		{Key: TagKeyName, Value: selection.substrate.name},
	}

	if selection.storage != (infrastructure.StorageClass{}) {
		tags = append(tags, Tag{Key: TagKeyStorage, Value: selection.storage.String()})
	}

	if len(selection.identities) > 0 {
		tags = append(tags, Tag{Key: TagKeyIdentities, Value: selection.identities.String()})
	}

	return tags
}

// Filter is the tag match a consuming casting writes into a data source.
func (selection Selection) Filter() map[string]string {
	return selection.match().Map()
}
