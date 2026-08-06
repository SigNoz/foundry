package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A fact is unqualified: the provider that stamps it decides the spelling, so
// nothing here may carry a prefix a provider's grammar could reject.
func TestTagKeys(t *testing.T) {
	tests := []struct {
		name        string
		tagKey      TagKey
		expectedKey string
	}{
		{name: "Name_Unqualified", tagKey: TagKeyName, expectedKey: "name"},
		{name: "Storage_Unqualified", tagKey: TagKeyStorage, expectedKey: "storage"},
		{name: "Identities_Unqualified", tagKey: TagKeyIdentities, expectedKey: "identities"},
		{name: "Owner_Unqualified", tagKey: TagKeyOwner, expectedKey: "owner"},
		{name: "SubnetType_Unqualified", tagKey: TagKeySubnetType, expectedKey: "subnet-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedKey, tt.tagKey.String())
			assert.NotContains(t, tt.tagKey.String(), "/")
			assert.NotContains(t, tt.tagKey.String(), ".")
		})
	}
}

// Two facts sharing a name would collapse into one tag whichever way a
// provider spells them.
func TestTagKeysAreDistinct(t *testing.T) {
	tagKeys := []TagKey{
		TagKeyName, TagKeyStorage, TagKeyIdentities,
		TagKeyOwner, TagKeySubnetType,
	}

	seen := make(map[string]struct{}, len(tagKeys))
	for _, tagKey := range tagKeys {
		assert.NotContains(t, seen, tagKey.String())
		seen[tagKey.String()] = struct{}{}
	}
}
