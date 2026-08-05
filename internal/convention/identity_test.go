package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIdentity(t *testing.T) {
	tests := []struct {
		name             string
		component        string
		ordinals         []int
		pass             bool
		expectedIdentity string
	}{
		{name: "ShardAndReplica_Valid", component: "telemetrystore", ordinals: []int{0, 0}, pass: true, expectedIdentity: "telemetrystore-0-0"},
		{name: "SingleOrdinal_Valid", component: "signoz", ordinals: []int{0}, pass: true, expectedIdentity: "signoz-0"},
		{name: "NoOrdinal_Valid", component: "metastore", pass: true, expectedIdentity: "metastore"},
		{name: "Empty_Invalid", component: "", ordinals: []int{0}, pass: false},
		{name: "ComponentWithSeparator_Invalid", component: "telemetry,store", ordinals: []int{0}, pass: false},
		{name: "NegativeOrdinal_Invalid", component: "signoz", ordinals: []int{-1}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := NewIdentity(tt.component, tt.ordinals...)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedIdentity, identity.String())
		})
	}
}

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name             string
		value            string
		pass             bool
		expectedIdentity Identity
	}{
		{name: "ShardAndReplica_Valid", value: "telemetrystore-0-0", pass: true, expectedIdentity: MustNewIdentity("telemetrystore", 0, 0)},
		{name: "SingleOrdinal_Valid", value: "signoz-0", pass: true, expectedIdentity: MustNewIdentity("signoz", 0)},
		{name: "NoOrdinal_Valid", value: "metastore", pass: true, expectedIdentity: MustNewIdentity("metastore")},
		{name: "HyphenatedComponent_Valid", value: "store-pool-1-2", pass: true, expectedIdentity: MustNewIdentity("store-pool", 1, 2)},
		{name: "Spaced_Trimmed", value: "  keeper-1  ", pass: true, expectedIdentity: MustNewIdentity("keeper", 1)},
		{name: "Empty_Invalid", value: "", pass: false},
		{name: "Blank_Invalid", value: "   ", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := ParseIdentity(tt.value)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedIdentity, identity)
		})
	}
}

// Parsing goes through NewIdentity, so a value a claim record could not have
// held is rejected on the way back in rather than becoming an identity nothing
// can match.
func TestParseIdentityDelegatesValidation(t *testing.T) {
	_, direct := NewIdentity("telemetry,store", 0)
	_, parsed := ParseIdentity("telemetry,store-0")

	assert.Error(t, direct)
	assert.Error(t, parsed)
}

func TestIdentitiesString(t *testing.T) {
	tests := []struct {
		name          string
		identities    Identities
		expectedValue string
	}{
		{
			name:          "Empty_RendersEmpty",
			identities:    Identities{},
			expectedValue: "",
		},
		{
			name:          "Single_RendersBare",
			identities:    Identities{MustNewIdentity("signoz", 0)},
			expectedValue: "signoz-0",
		},
		{
			name: "Several_JoinsSorted",
			identities: Identities{
				MustNewIdentity("telemetrystore", 0, 1),
				MustNewIdentity("metastore", 0),
				MustNewIdentity("telemetrystore", 0, 0),
			},
			expectedValue: "metastore-0,telemetrystore-0-0,telemetrystore-0-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedValue, tt.identities.String())
		})
	}
}

// Sorting is the whole reason the value is stable: the same claims stated in a
// different order have to render identically, or every plan shows a tag diff.
func TestIdentitiesRenderIndependentOfOrder(t *testing.T) {
	forward := Identities{MustNewIdentity("keeper", 0), MustNewIdentity("keeper", 1), MustNewIdentity("keeper", 2)}
	reversed := Identities{forward[2], forward[1], forward[0]}

	assert.Equal(t, forward.String(), reversed.String())
}

func TestParseIdentities(t *testing.T) {
	tests := []struct {
		name               string
		value              string
		pass               bool
		expectedIdentities Identities
	}{
		{
			name:               "Empty_YieldsNone",
			value:              "",
			pass:               true,
			expectedIdentities: nil,
		},
		{
			name:               "Single_YieldsOne",
			value:              "signoz-0",
			pass:               true,
			expectedIdentities: Identities{MustNewIdentity("signoz", 0)},
		},
		{
			name:               "Several_YieldsSorted",
			value:              "telemetrystore-0-1,metastore-0",
			pass:               true,
			expectedIdentities: Identities{MustNewIdentity("metastore", 0), MustNewIdentity("telemetrystore", 0, 1)},
		},
		{
			name:               "Spaced_TrimsEntries",
			value:              "keeper-0, keeper-1",
			pass:               true,
			expectedIdentities: Identities{MustNewIdentity("keeper", 0), MustNewIdentity("keeper", 1)},
		},
		{name: "TrailingSeparator_Invalid", value: "keeper-0,", pass: false},
		{name: "DoubledSeparator_Invalid", value: "keeper-0,,keeper-1", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identities, err := ParseIdentities(tt.value)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedIdentities, identities)
		})
	}
}

// The tag value written here is read back by Terraform's split(), so the two
// halves of the encoding have to agree. Round-tripping in Go is the only place
// that can be asserted.
func TestIdentitiesRoundTrip(t *testing.T) {
	identities := Identities{
		MustNewIdentity("telemetrykeeper", 0),
		MustNewIdentity("telemetrystore", 0, 0),
		MustNewIdentity("metastore", 0),
		MustNewIdentity("signoz", 0),
	}

	parsed, err := ParseIdentities(identities.String())

	assert.NoError(t, err)
	assert.Equal(t, identities.String(), parsed.String())
	assert.Len(t, parsed, len(identities))
}
