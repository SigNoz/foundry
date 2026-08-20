package config

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Config interface {
	// GetV1Alpha1 reads, dispatches, and validates every v1alpha1 casting
	// document the file holds, and returns them in cast order.
	GetV1Alpha1(ctx context.Context, path string) ([]v1alpha1.Machinery, error)

	// CreateOrUpdateV1Alpha1Lock replaces the casting's entry in the lock file,
	// keeping every other entry. A missing lock starts empty.
	CreateOrUpdateV1Alpha1Lock(ctx context.Context, machinery v1alpha1.Machinery, path string) error

	// PruneV1Alpha1Lock drops lock entries for castings the file no longer
	// declares; entries for declared castings are untouched. A lock left with
	// no entries is removed.
	PruneV1Alpha1Lock(ctx context.Context, declared []v1alpha1.Machinery, path string) error

	// GetV1Alpha1Lock reads the lock file from disk.
	GetV1Alpha1Lock(ctx context.Context, path string) ([]v1alpha1.Machinery, error)
}
