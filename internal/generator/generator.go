package generator

import (
	casting "github.com/signoz/foundry/internal/schema"
)

type Generator interface {
	GenerateComponent(config casting.Config) (map[string][]byte, error)
}