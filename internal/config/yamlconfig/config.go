package yamlconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/config"
	goyaml "gopkg.in/yaml.v3"
)

type yamlConfig struct {
}

func New() config.Config {
	return &yamlConfig{}
}

func (*yamlConfig) GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Casting, error) {
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
	// merge overrides into defaults (base)
	if err := v1alpha1.Merge(&defaultCasting, &casting); err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to merge default casting: %w", err)
	}

	return defaultCasting, nil
}

func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, config v1alpha1.Casting, path string) error {
	contents, err := goyaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	err = os.WriteFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"), contents, 0644)
	if err != nil {
		return fmt.Errorf("failed to write yaml file: %w", err)
	}

	return nil
}

func (*yamlConfig) GetV1Alpha1Lock(ctx context.Context, path string) (v1alpha1.Casting, error) {
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"))
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to read yaml file: %w", err)
	}

	var casting v1alpha1.Casting

	err = goyaml.Unmarshal(bytes, &casting)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return casting, nil
}
