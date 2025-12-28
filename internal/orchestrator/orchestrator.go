package orchestrator

import "cuelang.org/go/cue"

type Orchestrator interface {
	Platform() string
	Generate(config cue.Value, enabledComponents []string) (map[string][]byte, error)
}
