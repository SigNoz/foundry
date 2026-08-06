package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageClassUnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		pass     bool
		expected StorageClass
	}{
		{name: "Persistent_Valid", input: "persistent", pass: true, expected: StorageClassPersistent},
		{name: "Ephemeral_Valid", input: "ephemeral", pass: true, expected: StorageClassEphemeral},
		// An absent key never reaches the unmarshaler; an explicitly
		// empty one is a stated value that names no class.
		{name: "Empty_Invalid", input: "", pass: false},
		{name: "Unknown_Invalid", input: "durable", pass: false},
		{name: "Capitalised_Invalid", input: "Persistent", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := StorageClass{}
			err := class.UnmarshalText([]byte(tt.input))
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, class)

			// Round-trip: what the class renders unmarshals back to itself.
			roundTripped := StorageClass{}
			assert.NoError(t, roundTripped.UnmarshalText([]byte(class.String())))
			assert.Equal(t, class, roundTripped)
		})
	}
}

func TestStorageClassImplications(t *testing.T) {
	tests := []struct {
		name                       string
		class                      StorageClass
		expectedRequiresDataVolume bool
		expectedPinned             bool
	}{
		{name: "Persistent_CarriesDataAndIsPinned", class: StorageClassPersistent, expectedRequiresDataVolume: true, expectedPinned: true},
		{name: "Ephemeral_CarriesNothingAndScales", class: StorageClassEphemeral, expectedRequiresDataVolume: false, expectedPinned: false},
		{name: "Unset_CarriesNothingAndScales", class: StorageClass{}, expectedRequiresDataVolume: false, expectedPinned: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedRequiresDataVolume, tt.class.RequiresDataVolume())
			assert.Equal(t, tt.expectedPinned, tt.class.IsPinned())
		})
	}
}

func TestStorageClassEnum(t *testing.T) {
	assert.Equal(t, []any{"persistent", "ephemeral"}, StorageClassPersistent.Enum())
}
