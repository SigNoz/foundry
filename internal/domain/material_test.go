package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructuredMaterialCanBeTraversed(t *testing.T) {
	material, err := NewYAMLMaterial([]byte("service:\n  name: signoz\n"), "service.yaml")
	require.NoError(t, err)

	value, err := material.GetBytes("service.name")
	require.NoError(t, err)
	assert.Equal(t, "signoz", string(value))
}

func TestStructuredMaterialImplementations(t *testing.T) {
	yamlMaterial, err := NewYAMLMaterial([]byte("name: signoz\n"), "service.yaml")
	require.NoError(t, err)
	jsonMaterial, err := NewJSONMaterial([]byte(`{"name":"signoz"}`), "service.json")
	require.NoError(t, err)
	iniMaterial, err := NewINIMaterial([]byte("[Service]\nRestart=always\n"), "service.ini")
	require.NoError(t, err)

	assert.Implements(t, (*StructuredMaterial)(nil), yamlMaterial)
	assert.Implements(t, (*StructuredMaterial)(nil), jsonMaterial)
	assert.Implements(t, (*StructuredMaterial)(nil), iniMaterial)
}

func TestBlobMaterialStoresOpaqueContents(t *testing.T) {
	contents := []byte("FROM alpine\n")
	material := NewBlobMaterial(contents, "Dockerfile")

	assert.Equal(t, "Dockerfile", material.Path())
	assert.Equal(t, contents, material.FmtContents())
	assert.Implements(t, (*Material)(nil), material)
	assert.NotImplements(t, (*StructuredMaterial)(nil), material)
}

func TestStructuredMaterialFormatsContents(t *testing.T) {
	yamlMaterial, err := NewYAMLMaterial([]byte("service:\n  name: signoz\n"), "service.yaml")
	require.NoError(t, err)
	assert.Contains(t, string(yamlMaterial.FmtContents()), "service:")

	jsonMaterial, err := NewJSONMaterial([]byte(`{"service":{"name":"signoz"}}`), "service.json")
	require.NoError(t, err)
	assert.JSONEq(t, `{"service":{"name":"signoz"}}`, string(jsonMaterial.FmtContents()))

	iniMaterial, err := NewINIMaterial([]byte("[Service]\nRestart=always\n"), "service.ini")
	require.NoError(t, err)
	assert.Contains(t, string(iniMaterial.FmtContents()), "[Service]")
	assert.Contains(t, string(iniMaterial.FmtContents()), "Restart=always")
}

func TestYAMLMaterialDetectsMultiDoc(t *testing.T) {
	material, err := NewYAMLMaterial([]byte("---\nname: one\n---\nname: two\n"), "service.yaml")
	require.NoError(t, err)

	assert.True(t, material.IsMultiDoc())
}
