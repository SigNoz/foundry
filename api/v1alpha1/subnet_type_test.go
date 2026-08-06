package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubnetTypeUnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pass     bool
		expected SubnetType
	}{
		{name: "Private_Valid", input: "private", pass: true, expected: SubnetTypePrivate},
		{name: "Public_Valid", input: "public", pass: true, expected: SubnetTypePublic},
		// An absent key never reaches the unmarshaler; an explicitly empty one
		// is a stated value that names no subnetType.
		{name: "Empty_Invalid", input: "", pass: false},
		{name: "Unknown_Invalid", input: "internal", pass: false},
		{name: "Capitalised_Invalid", input: "Private", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subnetType := SubnetType{}
			err := subnetType.UnmarshalText([]byte(tt.input))
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, subnetType)

			// Round-trip: what the subnetType renders unmarshals back to itself.
			roundTripped := SubnetType{}
			assert.NoError(t, roundTripped.UnmarshalText([]byte(subnetType.String())))
			assert.Equal(t, subnetType, roundTripped)
		})
	}
}

func TestSubnetTypeImplications(t *testing.T) {
	tests := []struct {
		name           string
		subnetType     SubnetType
		expectedPublic bool
	}{
		{name: "Public_RoutesToTheInternet", subnetType: SubnetTypePublic, expectedPublic: true},
		{name: "Private_DoesNot", subnetType: SubnetTypePrivate, expectedPublic: false},
		{name: "Unset_DoesNot", subnetType: SubnetType{}, expectedPublic: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedPublic, tt.subnetType.IsPublic())
		})
	}
}

func TestSubnetTypeEnum(t *testing.T) {
	assert.Equal(t, []any{"private", "public"}, SubnetTypePrivate.Enum())
}
