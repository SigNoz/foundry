package generator

import (
	casting "github.com/SigNoz/foundry/internal/schema"
)

type Generator interface {
	GenerateComponent(config casting.Config) (map[string][]byte, error)
}