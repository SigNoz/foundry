package linux

import (
	"cuelang.org/go/cue"
)

type PlatformGenerator struct{}

// Generate linux platform-specific files.
func (g *PlatformGenerator) Generate(
	ctx *cue.Context,
	config cue.Value,
	enabledComponents map[string]bool,
) (cue.Value, map[string][]byte, error){

	return cue.Value{}, nil, nil
}