package patch

import (
	"context"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
)

// Patch applies a single patch entry to generated materials.
type Patch interface {
	// Apply applies a single patch entry to matching materials and returns the patched materials.
	Apply(ctx context.Context, materials []domain.Material, patch v1alpha1.PatchEntry) ([]domain.Material, error)
}

// MatchTarget checks if a material path matches a target pattern.
// Supports exact paths and glob patterns.
func MatchTarget(pattern, path string) (bool, error) {
	return filepath.Match(pattern, path)
}
