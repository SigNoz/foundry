package config

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Config interface {
	// Gets the v1alpha1 casting configuration from the given path.
	// It validates the configuration and returns an error if the configuration is invalid.
	GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Casting, error)

	// Creates the v1alpha1 casting lock file from the given configuration. This will have some support for checksumming in the future.
	CreateV1Alpha1Lock(ctx context.Context, config v1alpha1.Casting, path string) error

	// Gets the v1alpha1 lock file from the given path. This will validate the checksum of the configuration in the future.
	GetV1Alpha1Lock(ctx context.Context, path string) (v1alpha1.Casting, error)
}
