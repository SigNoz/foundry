package jsonpatch

import (
	"context"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newYAMLMaterial(t *testing.T, yamlContent string, path string) domain.Material {
	t.Helper()
	mat, err := domain.NewYAMLMaterial([]byte(yamlContent), path)
	require.NoError(t, err)
	return mat
}

func structuredMaterial(t *testing.T, material domain.Material) domain.StructuredMaterial {
	t.Helper()
	structured, ok := material.(domain.StructuredMaterial)
	require.True(t, ok, "expected structured material")
	return structured
}

func TestApply_ReplaceOperation(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `
services:
  clickhouse:
    mem_limit: "2G"
`, "docker-compose.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "docker-compose.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/services/clickhouse/mem_limit", Value: "4G"},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.NoError(t, err)
	require.Len(t, result, 1)

	contents := structuredMaterial(t, result[0]).FmtContents()
	assert.Contains(t, string(contents), "4G")
	assert.NotContains(t, string(contents), "2G")
}

func TestApply_AddOperation(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `
services:
  signoz:
    environment:
      - "EXISTING=true"
`, "docker-compose.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "docker-compose.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "add", Path: "/services/signoz/environment/-", Value: "CUSTOM_VAR=value"},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.NoError(t, err)

	contents := structuredMaterial(t, result[0]).FmtContents()
	assert.Contains(t, string(contents), "CUSTOM_VAR=value")
	assert.Contains(t, string(contents), "EXISTING=true")
}

func TestApply_RemoveOperation(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `
services:
  clickhouse:
    mem_limit: "2G"
    cpu_count: 4
`, "docker-compose.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "docker-compose.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "remove", Path: "/services/clickhouse/cpu_count"},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.NoError(t, err)

	contents := structuredMaterial(t, result[0]).FmtContents()
	assert.NotContains(t, string(contents), "cpu_count")
	assert.Contains(t, string(contents), "mem_limit")
}

func TestApply_GlobTarget(t *testing.T) {
	p := New()
	mat1 := newYAMLMaterial(t, `replicas: 1`, "clickhouse-shard-0.yaml")
	mat2 := newYAMLMaterial(t, `replicas: 1`, "clickhouse-shard-1.yaml")
	mat3 := newYAMLMaterial(t, `replicas: 1`, "signoz.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "clickhouse-shard-*.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/replicas", Value: 3},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat1, mat2, mat3}, pe)
	require.NoError(t, err)
	require.Len(t, result, 3)

	for _, i := range []int{0, 1} {
		contents := structuredMaterial(t, result[i]).FmtContents()
		assert.Contains(t, string(contents), "3")
	}

	contents := structuredMaterial(t, result[2]).FmtContents()
	assert.Contains(t, string(contents), "1")
}

func TestApply_FullPathMatch(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `name: test`, "deployment/compose.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "deployment/compose.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/name", Value: "patched"},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.NoError(t, err)

	contents := structuredMaterial(t, result[0]).FmtContents()
	assert.Contains(t, string(contents), "patched")
}

func TestApply_UnmatchedTargetReturnsError(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `name: test`, "test.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "nonexistent.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/name", Value: "patched"},
		},
	}

	_, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did not match any generated material")
}

func TestApply_InvalidPathReturnsError(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `name: test`, "test.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "test.yaml",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/nonexistent/path", Value: "value"},
		},
	}

	_, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply")
}

func TestApply_BlobMaterialReturnsError(t *testing.T) {
	p := New()
	mat := domain.NewBlobMaterial([]byte("FROM alpine\n"), "Dockerfile")

	pe := v1alpha1.PatchEntry{
		Target: "Dockerfile",
		Operations: []v1alpha1.PatchOperation{
			{Op: "replace", Path: "/from", Value: "ubuntu"},
		},
	}

	_, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json patch on blob material")
}

func TestApply_StructuredValue(t *testing.T) {
	p := New()
	mat := newYAMLMaterial(t, `
spec:
  template:
    spec:
      containers: []
`, "deployment.yaml")

	pe := v1alpha1.PatchEntry{
		Target: "deployment.yaml",
		Operations: []v1alpha1.PatchOperation{
			{
				Op:   "add",
				Path: "/spec/template/spec/tolerations",
				Value: []map[string]string{
					{"key": "dedicated", "value": "signoz", "effect": "NoSchedule"},
				},
			},
		},
	}

	result, err := p.Apply(context.Background(), []domain.Material{mat}, pe)
	require.NoError(t, err)

	contents := structuredMaterial(t, result[0]).FmtContents()
	assert.Contains(t, string(contents), "dedicated")
	assert.Contains(t, string(contents), "NoSchedule")
}
