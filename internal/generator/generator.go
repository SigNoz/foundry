package generator

import (
	"cuelang.org/go/cue"
)

// ComponentID identifies a component generator.
type ComponentID string

// PlatformID identifies a platform generator.
type PlatformID string

// ComponentGenerator generates files for a specific component (e.g., clickhouse, signoz).
type ComponentGenerator interface {
	GenerateComponent(config cue.Value) (map[string][]byte, error)
}

// PlatformGenerator generates files for a specific component (e.g., linux).
type PlatformGenerator interface {
	Generate(ctx *cue.Context, config cue.Value, enabledComponents map[string]bool) (cue.Value, map[string][]byte, error)
}