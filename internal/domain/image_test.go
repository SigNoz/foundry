package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImage(t *testing.T) {
	tests := []struct {
		name               string
		raw                string
		expectedRepository string
		expectedTag        string
		pass               bool
	}{
		{"RepositoryAndTag_Valid", "clickhouse/clickhouse-server:25.12.5", "clickhouse/clickhouse-server", "25.12.5", true},
		{"FlavoredTag_Valid", "clickhouse/clickhouse-server:25.12.5-alpine", "clickhouse/clickhouse-server", "25.12.5-alpine", true},
		{"NoTag_DefaultsToLatest", "signoz/signoz-otel-collector", "signoz/signoz-otel-collector", "latest", true},
		{"RegistryPort_NotMistakenForTag", "host:5000/clickhouse", "host:5000/clickhouse", "latest", true},
		{"Empty_Invalid", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := ParseImage(tt.raw)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRepository, image.Repository())
			assert.Equal(t, tt.expectedTag, image.Tag())
		})
	}
}

func TestImageVersion(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		expectedOK bool
	}{
		{"SemverTag_Resolves", "clickhouse/clickhouse-server:25.12.5", true},
		{"FlavoredTag_Resolves", "clickhouse/clickhouse-server:25.12.5-alpine", true},
		{"LatestTag_Unknown", "signoz/signoz-otel-collector:latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := ParseImage(tt.raw)
			assert.NoError(t, err)
			_, ok := image.Version()
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestImageWithTag(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		tag        string
		newTag     string
		expected   string
	}{
		{"Tag_Replaced", "clickhouse/clickhouse-server", "25.5.6", "25.12.5", "clickhouse/clickhouse-server:25.12.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image := MustNewImage(tt.repository, tt.tag)
			assert.Equal(t, tt.expected, image.WithTag(tt.newTag).String())
		})
	}
}
