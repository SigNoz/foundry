package generator

import (
	"cuelang.org/go/cue"
)

type Generator interface {
	GenerateComponent(config cue.Value) (map[string][]byte, error)
}