package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSubnetType(t *testing.T) {
	tests := []struct {
		name             string
		value            string
		pass             bool
		expectedPublic   bool
		expectedSpelling string
	}{
		{name: "Private_Valid", value: "private", pass: true, expectedSpelling: "private"},
		{name: "Public_Valid", value: "public", pass: true, expectedPublic: true, expectedSpelling: "public"},
		{name: "Empty_Invalid", value: "", pass: false},
		{name: "Unknown_Invalid", value: "internal", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subnetType, err := ParseSubnetType(tt.value)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedSpelling, subnetType.String())
			assert.Equal(t, tt.expectedPublic, subnetType.IsPublic())
		})
	}
}
