package convention

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseZone(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		pass          bool
		expectedZone  string
		expectedShort string
	}{
		{name: "AWSZone_Valid", input: "us-east-1a", pass: true, expectedZone: "us-east-1a", expectedShort: "east1a"},
		{name: "GCPZone_Valid", input: "asia-south2-c", pass: true, expectedZone: "asia-south2-c", expectedShort: "south2c"},
		{name: "GCPRegionZone_Valid", input: "us-central1-b", pass: true, expectedZone: "us-central1-b", expectedShort: "central1b"},
		{name: "GovCloudZone_Valid", input: "us-gov-east-1a", pass: true, expectedZone: "us-gov-east-1a", expectedShort: "goveast1a"},
		{name: "Empty_Invalid", input: "", pass: false},
		{name: "NoSeparator_Invalid", input: "useast1a", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, err := ParseZone(tt.input)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// The provider form is kept verbatim; only the short form is derived.
			assert.Equal(t, tt.expectedZone, zone.String())
			assert.Equal(t, tt.expectedShort, zone.Short())
		})
	}
}
