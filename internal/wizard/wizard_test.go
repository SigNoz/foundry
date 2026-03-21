package wizard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCastingFromFlags(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		mode        string
		flavor      string
		platform    string
		output      string
		expectError bool
	}{
		{
			name:     "docker compose with all flags",
			flagName: "my-signoz",
			mode:     "docker",
			flavor:   "compose",
			output:   "custom.yaml",
		},
		{
			name:     "platform-based deployment",
			platform: "render",
			flavor:   "blueprint",
		},
		{
			name:     "defaults for empty name and output",
			mode:     "docker",
			flavor:   "compose",
			flagName: "",
			output:   "",
		},
		{
			name:        "missing mode and platform",
			flavor:      "compose",
			expectError: true,
		},
		{
			name:        "missing flavor",
			mode:        "docker",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := BuildCastingFromFlags(tc.flagName, tc.mode, tc.flavor, tc.platform, tc.output)
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)

			if tc.flagName != "" {
				assert.Equal(t, tc.flagName, result.Name)
			} else {
				assert.Equal(t, "signoz", result.Name)
			}

			if tc.output != "" {
				assert.Equal(t, tc.output, result.OutputPath)
			} else {
				assert.Equal(t, "casting.yaml", result.OutputPath)
			}
		})
	}
}

func TestWriteCasting(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "casting.yaml")

	result := &Result{
		Name: "test-signoz",
		Deployment: v1alpha1.TypeDeployment{
			Mode:   "docker",
			Flavor: "compose",
		},
		OutputPath: outputPath,
	}

	err := WriteCasting(result)
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "name: test-signoz")
	assert.Contains(t, content, "mode: docker")
	assert.Contains(t, content, "flavor: compose")
	assert.Contains(t, content, "apiVersion: v1alpha1")
}

func TestBuildDeploymentOptions(t *testing.T) {
	deployments := []v1alpha1.TypeDeployment{
		{Mode: "docker", Flavor: "compose"},
		{Platform: "render", Flavor: "blueprint"},
		{Mode: "kubernetes", Flavor: "helm"},
	}

	options := buildDeploymentOptions(deployments)
	assert.Len(t, options, 3)

	// Should be sorted by label
	for i := 1; i < len(options); i++ {
		assert.True(t, options[i-1].Label <= options[i].Label, "options should be sorted")
	}
}

func TestFormatDeploymentLabel(t *testing.T) {
	tests := []struct {
		d        v1alpha1.TypeDeployment
		expected string
	}{
		{v1alpha1.TypeDeployment{Mode: "docker", Flavor: "compose"}, "docker / compose"},
		{v1alpha1.TypeDeployment{Platform: "render", Flavor: "blueprint"}, "render / blueprint"},
		{v1alpha1.TypeDeployment{Platform: "aws", Mode: "kubernetes", Flavor: "helm"}, "aws / kubernetes / helm"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, formatDeploymentLabel(tc.d))
	}
}
