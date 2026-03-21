package ux

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		expected string
	}{
		{"milliseconds", 150 * time.Millisecond, "(150ms)"},
		{"sub-second", 999 * time.Millisecond, "(999ms)"},
		{"one second", 1 * time.Second, "(1.0s)"},
		{"seconds", 2500 * time.Millisecond, "(2.5s)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatDuration(tc.d))
		})
	}
}

func TestNewUX(t *testing.T) {
	// Test that UX can be created in both modes without panicking
	u := New(false)
	assert.NotNil(t, u)
	assert.NotNil(t, u.Logger())

	u = New(true)
	assert.NotNil(t, u)
	assert.NotNil(t, u.Logger())
}
