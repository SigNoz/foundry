package convention

import (
	"github.com/signoz/foundry/internal/domain"
)

// TagKey is a tag's key. Which of these a consumer filters on is decided by
// Selection, not declared here.
type TagKey struct {
	key string
}

var (
	TagKeyName         = TagKey{key: domain.MetadataPrefix + "name"}
	TagKeyStorage      = TagKey{key: domain.MetadataPrefix + "storage"}
	TagKeyIdentities   = TagKey{key: domain.MetadataPrefix + "identities"}
	TagKeyResourceKind = TagKey{key: domain.MetadataPrefix + "resource-kind"}
	TagKeyOwner        = TagKey{key: domain.MetadataPrefix + "owner"}
	TagKeyVisibility   = TagKey{key: domain.MetadataPrefix + "visibility"}

	// TagKeyDisplayName is unprefixed: "Name" is the provider's own convention
	// for what a console shows.
	TagKeyDisplayName = TagKey{key: "Name"}
)

func (tagKey TagKey) String() string {
	return tagKey.key
}

type Tag struct {
	Key   TagKey
	Value string
}

type Tags []Tag

func (tags Tags) Map() map[string]string {
	out := make(map[string]string, len(tags))
	for _, tag := range tags {
		out[tag.Key.String()] = tag.Value
	}

	return out
}
