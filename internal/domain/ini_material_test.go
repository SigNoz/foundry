package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewINIMaterial(t *testing.T) {
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
			name:                         "OneSection_Valid",
			contents:                     []byte("[Service]\nRestart=always\nEnvironment=SIGNOZ=1\nEnvironment=OTEL=1\n"),
			path:                         "service.ini",
			pass:                         true,
			expectedJSON:                 []byte(`{"Service":{"Restart":"always","Environment":["SIGNOZ=1","OTEL=1"]}}`),
			expectedFmtContents:          []byte("[Service]\nEnvironment=SIGNOZ=1\nEnvironment=OTEL=1\nRestart=always\n"),
			expectedHasMultipleDocuments: false,
		},
		{
			name:                         "TwoSections_Valid",
			contents:                     []byte("[Service]\nRestart=always\nEnvironment=SIGNOZ=1\nEnvironment=OTEL=1\n[Database]\nHost=localhost\nPort=5432\n"),
			path:                         "service.ini",
			pass:                         true,
			expectedJSON:                 []byte(`{"Service":{"Restart":"always","Environment":["SIGNOZ=1","OTEL=1"]},"Database":{"Host":"localhost","Port":"5432"}}`),
			expectedFmtContents:          []byte("[Database]\nHost=localhost\nPort=5432\n\n[Service]\nEnvironment=SIGNOZ=1\nEnvironment=OTEL=1\nRestart=always\n"),
			expectedHasMultipleDocuments: false,
		},
		{
			name:     "Invalid",
			contents: []byte("[Service\nRestart=always\n"),
			path:     "service.ini",
			pass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material, err := NewINIMaterial(tt.contents, tt.path)
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

func TestINIMaterialGetBytes(t *testing.T) {
	tests := []struct {
		name     string
		material INIMaterial
		path     string
		pass     bool
		expected []byte
	}{
		{
			name:     "OneSection_Exists",
			material: MustNewINIMaterial([]byte("[Service]\nRestart=always\nEnvironment=SIGNOZ=1\nEnvironment=OTEL=1\n"), "service.ini"),
			path:     "Service.Restart",
			pass:     true,
			expected: []byte("always"),
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
