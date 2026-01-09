// Package loader package provides functionality to load, validate, and unify
package loader

import (
	"context"

	"github.com/signoz/foundry/api/v1alpha1"
)

type Loader interface {
	// LoadV1Alpha1 loads the v1alpha1 casting configuration from the given path. It also validates the configuration.
	LoadV1Alpha1(ctx context.Context, path string) (v1alpha1.Casting, error)
}
