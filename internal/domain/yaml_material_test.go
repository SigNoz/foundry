package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewYAMLMaterial(t *testing.T) {
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
			name:                         "SingleDocument_Valid",
			contents:                     []byte("service:\n  names:\n    - query-service\n    - frontend\n"),
			path:                         "service.yaml",
			pass:                         true,
			expectedJSON:                 []byte(`{"service":{"names":["query-service","frontend"]}}`),
			expectedFmtContents:          []byte("service:\n  names:\n  - query-service\n  - frontend\n"),
			expectedHasMultipleDocuments: false,
		},
		{
			name:                         "MultiDocument_Valid",
			contents:                     []byte("---\nname: one\n---\nname: two\n"),
			path:                         "service.yaml",
			pass:                         true,
			expectedJSON:                 []byte(`[{"name":"one"},{"name":"two"}]`),
			expectedFmtContents:          []byte("name: one\n---\nname: two\n"),
			expectedHasMultipleDocuments: true,
		},
		{
			name:     "Invalid",
			contents: []byte("service: [unterminated\n"),
			path:     "service.yaml",
			pass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := NewYAMLMaterial(tt.contents, tt.path)
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

func TestYAMLMaterialGetBytes(t *testing.T) {
	tests := []struct {
		name     string
		material YAMLMaterial
		path     string
		pass     bool
		expected []byte
	}{
		{
			name:     "SingleDocument_Exists",
			material: MustNewYAMLMaterial([]byte("service:\n  name: query-service\n"), "service.yaml"),
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
