package aws

import (
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/convention"
	"github.com/stretchr/testify/assert"
)

// A filter's keys are the only ones whose spelling live infrastructure depends
// on: renaming one leaves it unmatched, with no checkpoint to catch it. AWS is
// the only provider whose tag keys accept this prefix, which is why the
// spelling is asserted here and not beside the facts.
func TestFilterKeysMatchDeployedSpelling(t *testing.T) {
	filter := Filter(convention.MustNewSubstrate("foundry").Select().
		WithSubnetType(v1alpha1.SubnetTypePrivate).
		WithStorage(v1alpha1.StorageClassPersistent).
		WithClaims(convention.Identities{convention.MustNewIdentity("signoz", 0)}))

	assert.Equal(t, map[string]string{
		"foundry.signoz.io/name":        "foundry",
		"foundry.signoz.io/subnet-type": "private",
		"foundry.signoz.io/storage":     "persistent",
		"foundry.signoz.io/identities":  "signoz-0",
	}, filter)
}

// The display tag is the provider's own, not foundry's, so it carries no prefix.
func TestDisplayNameIsProviderNative(t *testing.T) {
	assert.Equal(t, "Name", displayName)
	assert.Equal(t, "foundry.signoz.io/owner", Tag(convention.TagKeyOwner))
}
