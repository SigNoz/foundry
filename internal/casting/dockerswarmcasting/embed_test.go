package dockerswarmcasting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPServiceInheritsItsImageHealthcheck(t *testing.T) {
	source, err := templates.ReadFile("templates/compose.yaml.gotmpl")
	assert.NoError(t, err)
	assert.NotContains(t, string(source), "- \"/usr/local/bin/signoz-mcp-server\"\n      - \"healthcheck\"")
	assert.NotContains(t, string(source), "- \"http://localhost:8000/livez\"")
}
