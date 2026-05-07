package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJSONMaterial(t *testing.T) {
	tests := []struct {
		name                         string
		contents                     []byte
		path                         string
		pass                         bool
		expectedJSON                 []byte
		expectedFmtContents          []byte
		expectedHasMultipleDocuments bool
	}{
		{
			name:                         "Valid",
			contents:                     []byte(`{"service":{"names":["query-service","frontend"]}}`),
			path:                         "service.json",
			pass:                         true,
			expectedJSON:                 []byte(`{"service":{"names":["query-service","frontend"]}}`),
			expectedFmtContents:          []byte(`{"service":{"names":["query-service","frontend"]}}`),
			expectedHasMultipleDocuments: false,
		},
		{
			name:     "Invalid",
			contents: []byte(`{"service":`),
			path:     "service.json",
			pass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := NewJSONMaterial(tt.contents, tt.path)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.path, material.Path())
			assert.Equal(t, tt.expectedHasMultipleDocuments, material.HasMultipleDocuments())
			assert.JSONEq(t, string(tt.expectedJSON), string(material.JSONContents()))
			assert.Equal(t, tt.expectedFmtContents, material.FmtContents())
		})
	}
}

func TestJSONMaterialGetBytes(t *testing.T) {
	tests := []struct {
		name     string
		material JSONMaterial
		path     string
		pass     bool
		expected []byte
	}{
		{
			name:     "Exists",
			material: MustNewJSONMaterial([]byte(`{"service":{"name":"query-service"}}`), "service.json"),
			path:     "service.name",
			pass:     true,
			expected: []byte("query-service"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tt.material.GetBytes(tt.path)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.Equal(t, tt.expected, output)
		})
	}
}
