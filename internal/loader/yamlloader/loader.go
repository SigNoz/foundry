package yamlloader

import (
	"context"
	"fmt"
	"os"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/loader"
	goyaml "gopkg.in/yaml.v3"
)

var _ loader.Loader = (*yamlLoader)(nil)

type yamlLoader struct {
}

func New() *yamlLoader {
	return &yamlLoader{}
}

func (loader *yamlLoader) LoadV1Alpha1(ctx context.Context, path string) (v1alpha1.Casting, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to read yaml file: %w", err)
	}

	var casting v1alpha1.Casting

	err = goyaml.Unmarshal(bytes, &casting)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	defaultCasting := v1alpha1.DefaultCasting()

	if err := v1alpha1.Merge(&casting, &defaultCasting); err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to merge default casting: %w", err)
	}

	return casting, nil
}
