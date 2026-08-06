package convention

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// The key names the group and the class selects it: two groups may share a
// class, and a consumer that filters on the class reaches both.
func TestNodeGroup(t *testing.T) {
	hot := NewNodeGroup(MustNewKey("hot"), v1alpha1.StorageClassPersistent)
	cold := NewNodeGroup(MustNewKey("cold"), v1alpha1.StorageClassPersistent)

	assert.Equal(t, "hot", hot.Key().String())
	assert.Equal(t, "cold", cold.Key().String())
	assert.Equal(t, hot.Storage(), cold.Storage())

	filter := MustNewSubstrate("foundry").Select().WithStorage(v1alpha1.StorageClassPersistent).Match()
	assert.Equal(t, map[TagKey]string{
		TagKeyName:    "foundry",
		TagKeyStorage: "persistent",
	}, filter)
}
