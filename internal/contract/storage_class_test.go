package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStorageClass(t *testing.T) {
	tests := []struct {
		name             string
		value            string
		pass             bool
		expectedPinned   bool
		expectedDataVol  bool
		expectedSpelling string
	}{
		{name: "Persistent_Valid", value: "persistent", pass: true, expectedPinned: true, expectedDataVol: true, expectedSpelling: "persistent"},
		{name: "Ephemeral_Valid", value: "ephemeral", pass: true, expectedSpelling: "ephemeral"},
		{name: "Empty_Invalid", value: "", pass: false},
		{name: "Unknown_Invalid", value: "durable", pass: false},
		{name: "WrongCase_Invalid", value: "Persistent", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, err := ParseStorageClass(tt.value)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSpelling, class.String())
			assert.Equal(t, tt.expectedPinned, class.IsPinned())
			assert.Equal(t, tt.expectedDataVol, class.RequiresDataVolume())
		})
	}
}
