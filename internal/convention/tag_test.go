package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagKeys(t *testing.T) {
	tests := []struct {
		name        string
		tagKey      TagKey
		expectedKey string
	}{
		{name: "Name_Prefixed", tagKey: TagKeyName, expectedKey: "foundry.signoz.io/name"},
		{name: "Storage_Prefixed", tagKey: TagKeyStorage, expectedKey: "foundry.signoz.io/storage"},
		{name: "Identities_Prefixed", tagKey: TagKeyIdentities, expectedKey: "foundry.signoz.io/identities"},
		{name: "ResourceKind_Prefixed", tagKey: TagKeyResourceKind, expectedKey: "foundry.signoz.io/resource-kind"},
		{name: "Owner_Prefixed", tagKey: TagKeyOwner, expectedKey: "foundry.signoz.io/owner"},
		{name: "Visibility_Prefixed", tagKey: TagKeyVisibility, expectedKey: "foundry.signoz.io/visibility"},
		{name: "DisplayName_ProviderNative", tagKey: TagKeyDisplayName, expectedKey: "Name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedKey, tt.tagKey.String())
		})
	}
}

// Two keys sharing a string would collide on one resource, the second silently
// overwriting the first.
func TestTagKeysAreDistinct(t *testing.T) {
	tagKeys := []TagKey{
		TagKeyName, TagKeyStorage, TagKeyIdentities,
		TagKeyResourceKind, TagKeyOwner, TagKeyVisibility, TagKeyDisplayName,
	}

	seen := make(map[string]struct{}, len(tagKeys))
	for _, tagKey := range tagKeys {
		assert.NotContains(t, seen, tagKey.String())
		seen[tagKey.String()] = struct{}{}
	}
}
