package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		pass        bool
		expectedKey string
	}{
		{name: "Word_Valid", input: "persistent", pass: true, expectedKey: "persistent"},
		{name: "InteriorHyphens_Valid", input: "private-us-east-1a", pass: true, expectedKey: "private-us-east-1a"},
		{name: "Digits_Valid", input: "pool2", pass: true, expectedKey: "pool2"},
		{name: "Empty_Invalid", input: "", pass: false},
		{name: "Uppercase_Invalid", input: "Private", pass: false},
		{name: "LeadingHyphen_Invalid", input: "-private", pass: false},
		{name: "TrailingHyphen_Invalid", input: "private-", pass: false},
		{name: "Underscore_Invalid", input: "private_a", pass: false},
		{name: "Dot_Invalid", input: "private.a", pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := NewKey(tt.input)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedKey, key.String())
		})
	}
}

// A key reaches a derived name unchanged, so what it accepts is what a name
// segment may contain.
func TestKeyAcceptsWhatASubstrateNameDoes(t *testing.T) {
	for _, name := range []string{"foundry", "signoz-prod-eu", "signoz2"} {
		key, err := NewKey(name)

		assert.NoError(t, err)
		assert.Equal(t, name, key.String())
	}
}
