package domain

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		pass bool
	}{
		{"Plain_Valid", "25.12.5", true},
		{"Flavored_Valid", "25.12.5-alpine", true},
		{"Distroless_Valid", "25.12.5-distroless", true},
		{"FourPart_Valid", "25.12.5.44", true},
		{"FourPartFlavored_Valid", "25.12.5.44-alpine", true},
		{"Latest_Invalid", "latest", false},
		{"Empty_Invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := ParseVersion(tt.raw)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.raw, version.String())
		})
	}
}

func TestVersionSatisfies(t *testing.T) {
	constraint, err := semver.NewConstraint(">=25.12.5")
	assert.NoError(t, err)

	tests := []struct {
		name     string
		raw      string
		expected bool
	}{
		{"Equal_Satisfies", "25.12.5", true},
		{"FlavoredEqual_Satisfies", "25.12.5-alpine", true},
		{"FourPartEqual_Satisfies", "25.12.5.44", true},
		{"Newer_Satisfies", "25.13.0", true},
		{"Older_DoesNotSatisfy", "25.5.6", false},
		{"OlderFlavored_DoesNotSatisfy", "25.5.6-alpine", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, MustParseVersion(tt.raw).Satisfies(constraint))
		})
	}
}
