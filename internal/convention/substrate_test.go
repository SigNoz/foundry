package convention

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubstrate(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		pass         bool
		expectedName string
	}{
		{name: "Lowercase_Valid", input: "foundry", pass: true, expectedName: "foundry"},
		{name: "InteriorHyphens_Valid", input: "signoz-prod-eu", pass: true, expectedName: "signoz-prod-eu"},
		{name: "Digits_Valid", input: "signoz2", pass: true, expectedName: "signoz2"},
		{name: "Empty_Invalid", input: "", pass: false},
		{name: "Uppercase_Invalid", input: "Foundry", pass: false},
		{name: "LeadingHyphen_Invalid", input: "-foundry", pass: false},
		{name: "TrailingHyphen_Invalid", input: "foundry-", pass: false},
		{name: "Underscore_Invalid", input: "foundry_prod", pass: false},
		{name: "TooLong_Invalid", input: strings.Repeat("a", maxNameLength+1), pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			substrate, err := NewSubstrate(tt.input)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedName, substrate.String())
		})
	}
}
