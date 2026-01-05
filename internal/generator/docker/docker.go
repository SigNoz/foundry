package docker

import (
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/instrumentation"
	"github.com/signoz/foundry/internal/loader"
)

type PlatformGenerator struct{}

// Generate docker platform-specific files.
func (g *PlatformGenerator) Generate(
	ctx *cue.Context,
	config cue.Value,
	enabledComponents map[string]bool,
) (cue.Value, map[string][]byte, error) {
	logger := instrumentation.NewLogger(true).With("platform.generator", "docker")
	logger.Debug("Starting Docker platform generation")

	// Generate a map of versions for enabled components
	componentVersions := make(map[string]string)
	for k := range enabledComponents {
		versionValue, err := getValue(config, fmt.Sprintf("components.%s.version", k))
		if err != nil {
			return cue.Value{}, nil, fmt.Errorf("failed to get version for component %s: %w", k, err)
		}
		versionStr, _ := versionValue.String()
		componentVersions[k] = versionStr
	}
	fmt.Println("Component Versions:", componentVersions)

	// Read the Docker compose schema
	deployment, err := loader.LoadSchema(ctx, "castings/docker/docker.cue")
	if err != nil {
		return cue.Value{}, nil, errors.New("schema compilation error: " + err.Error())
	}

	// Iterate over the component versions to merge with deployment lookups
	for component, version := range componentVersions {
		versionValue := ctx.Encode(version)
		deployment = mergeValues(deployment, fmt.Sprintf("#Params.%s_VERSION", strings.ToUpper(component)), versionValue)
	}

	return deployment, nil, nil
}

// getValue retrieves a value from the CUE configuration based on the provided path.
func getValue(c cue.Value, path string) (cue.Value, error) {
	return c.LookupPath(cue.ParsePath(path)), nil
}

// NOTE: To be used for other configurations
func mergeValues(c cue.Value, path string, value cue.Value) cue.Value {
	return c.FillPath(cue.ParsePath(path), value)
}
