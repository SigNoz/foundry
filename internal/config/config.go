package config

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Config interface {
	// GetV1Alpha1 reads, dispatches, and validates every v1alpha1 casting
	// document the file holds, and returns them in cast order.
	GetV1Alpha1(ctx context.Context, path string) ([]v1alpha1.Machinery, error)

	// CreateV1Alpha1Lock writes the resolved castings to the lock file.
	CreateV1Alpha1Lock(ctx context.Context, machineries []v1alpha1.Machinery, path string) error

	// GetV1Alpha1Lock reads the lock file from disk.
	GetV1Alpha1Lock(ctx context.Context, path string) ([]v1alpha1.Machinery, error)

	// MatchV1Alpha1Lock reports whether resolved castings match the lock on disk.
	MatchV1Alpha1Lock(ctx context.Context, machineries []v1alpha1.Machinery, path string) (bool, error)

	// DeleteV1Alpha1Lock removes the lock file from disk.
	DeleteV1Alpha1Lock(ctx context.Context, path string) error
}
