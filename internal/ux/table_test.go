package ux

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestTableRender(t *testing.T) {
	// Disable colors for predictable test output
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = true })

	table := NewTable("Name", "Status")
	table.AddRow("docker", "available")
	table.AddRow("helm", "missing")

	var buf bytes.Buffer
	table.Render(&buf)
	output := buf.String()

	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "docker")
	assert.Contains(t, output, "available")
	assert.Contains(t, output, "helm")
	assert.Contains(t, output, "missing")
	assert.Contains(t, output, "───")
}

func TestTableEmptyHeaders(t *testing.T) {
	table := NewTable()
	var buf bytes.Buffer
	table.Render(&buf)
	assert.Empty(t, buf.String())
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{5400, "5.3 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, formatBytes(tc.input))
	}
}
