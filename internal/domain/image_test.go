package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImage(t *testing.T) {
	tests := []struct {
		name               string
		raw                string
		expectedRegistry   string
		expectedRepository string
		expectedTag        string
		expectedString     string
		pass               bool
	}{
		{"RepositoryAndTag_Valid", "clickhouse/clickhouse-server:25.12.5", "", "clickhouse/clickhouse-server", "25.12.5", "clickhouse/clickhouse-server:25.12.5", true},
		{"FlavoredTag_Valid", "clickhouse/clickhouse-server:25.12.5-alpine", "", "clickhouse/clickhouse-server", "25.12.5-alpine", "clickhouse/clickhouse-server:25.12.5-alpine", true},
		{"NoTag_DefaultsToLatest", "signoz/signoz-otel-collector", "", "signoz/signoz-otel-collector", "latest", "signoz/signoz-otel-collector:latest", true},
		{"BareName_NoRegistry", "postgres:16", "", "postgres", "16", "postgres:16", true},
		{"Namespace_NotRegistry", "signoz/signoz:v1", "", "signoz/signoz", "v1", "signoz/signoz:v1", true},
		{"Registry_Split", "ghcr.io/signoz/signoz:v1", "ghcr.io", "signoz/signoz", "v1", "ghcr.io/signoz/signoz:v1", true},
		{"RegistryPort_NotMistakenForTag", "host:5000/clickhouse", "host:5000", "clickhouse", "latest", "host:5000/clickhouse:latest", true},
		{"RegistryPortAndTag_Valid", "myreg.example.com:5000/signoz/signoz:v1", "myreg.example.com:5000", "signoz/signoz", "v1", "myreg.example.com:5000/signoz/signoz:v1", true},
		{"Localhost_IsRegistry", "localhost/signoz:v1", "localhost", "signoz", "v1", "localhost/signoz:v1", true},
		{"Empty_Invalid", "", "", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image, err := ParseImage(tt.raw)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRegistry, image.Registry())
			assert.Equal(t, tt.expectedRepository, image.Repository())
			assert.Equal(t, tt.expectedTag, image.Tag())
			assert.Equal(t, tt.expectedString, image.String())
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
		{"Registry_Kept", "ghcr.io/signoz/signoz", "v1", "v2", "ghcr.io/signoz/signoz:v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image := MustNewImage(tt.repository, tt.tag)
			assert.Equal(t, tt.expected, image.WithTag(tt.newTag).String())
		})
	}
}
