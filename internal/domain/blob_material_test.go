package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBlobMaterial(t *testing.T) {
	tests := []struct {
		name                string
		contents            []byte
		path                string
		expectedFmtContents []byte
	}{
		{
			name:                "Dockerfile",
			contents:            []byte("FROM alpine\nRUN echo '{not-json}'\n"),
			path:                "Dockerfile",
			expectedFmtContents: []byte("FROM alpine\nRUN echo '{not-json}'\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			material := NewBlobMaterial(tt.contents, tt.path)

			assert.Equal(t, tt.path, material.Path())
			assert.Equal(t, tt.expectedFmtContents, material.FmtContents())
		})
	}
}
