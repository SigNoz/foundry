package yamlconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

type yamlConfig struct {
	v1alphaSchema *jsonschema.Resolved
}

func New() config.Config {
	return &yamlConfig{
		v1alphaSchema: v1alpha1.JSONSchema(),
	}
}

func (config *yamlConfig) GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Casting, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to read yaml file: %w", err)
	}

	var casting v1alpha1.Casting

	err = domain.UnmarshalYAML(bytes, &casting)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	defaultCasting := v1alpha1.DefaultCasting()
	// merge overrides into defaults (base)
	if err := v1alpha1.Merge(&defaultCasting, &casting); err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to merge default casting: %w", err)
	}

	contents, err := json.Marshal(defaultCasting)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to marshal default casting: %w", err)
	}

	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to unmarshal default casting into map[string]any for validation: %w", err)
	}

	// validate the casting against the schema
	if err := config.v1alphaSchema.Validate(toValidate); err != nil {
		return v1alpha1.Casting{}, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s", path)
	}

	return defaultCasting, nil
}

func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, config v1alpha1.Casting, path string) error {
	contents, err := domain.MarshalYAML(config)
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

	err = domain.UnmarshalYAML(bytes, &casting)
	if err != nil {
		return v1alpha1.Casting{}, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return casting, nil
}
