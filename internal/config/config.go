package config

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Config interface {
	// GetV1Alpha1 reads, dispatches, and validates a v1alpha1 casting from disk.
	GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Machinery, error)

	// CreateV1Alpha1Lock writes the resolved casting to the lock file.
	CreateV1Alpha1Lock(ctx context.Context, machinery v1alpha1.Machinery, path string) error

	// GetV1Alpha1Lock reads the lock file from disk.
	GetV1Alpha1Lock(ctx context.Context, path string) (v1alpha1.Machinery, error)
}
