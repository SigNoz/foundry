package yamlconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

type yamlConfig struct{}

func New() config.Config {
	return &yamlConfig{}
}

// GetV1Alpha1 reads, peeks at kind, dispatches to the per-Kind loader, merges
// defaults, validates against the per-Kind schema, and returns the resolved
// casting wrapped as v1alpha1.Machinery.
func (*yamlConfig) GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Machinery, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml file: %w", err)
	}

	kind, err := peekKind(bytes)
	if err != nil {
		return nil, err
	}

	switch kind {
	case v1alpha1.KindInstallation:
		return loadInstallation(bytes, path)
	}
	return nil, errors.Newf(errors.TypeUnsupported, "unknown casting kind %q", kind)
}

// peekKind decodes only the kind field from raw bytes. Empty or missing kind
// defaults to KindInstallation so existing castings without `kind` keep working.
func peekKind(bytes []byte) (v1alpha1.Kind, error) {
	var probe struct {
		Kind v1alpha1.Kind `json:"kind" yaml:"kind"`
	}
	if err := domain.UnmarshalYAML(bytes, &probe); err != nil {
		return v1alpha1.Kind{}, fmt.Errorf("failed to peek kind: %w", err)
	}
	if probe.Kind == (v1alpha1.Kind{}) {
		return v1alpha1.KindInstallation, nil
	}
	return probe.Kind, nil
}

func loadInstallation(bytes []byte, path string) (v1alpha1.Machinery, error) {
	var loaded installation.Casting
	if err := domain.UnmarshalYAML(bytes, &loaded); err != nil {
		return nil, fmt.Errorf("failed to unmarshal installation casting: %w", err)
	}

	base := installation.Default()
	if err := v1alpha1.Merge(base, &loaded); err != nil {
		return nil, fmt.Errorf("failed to merge default installation casting: %w", err)
	}

	contents, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal installation casting: %w", err)
	}
	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal installation casting for validation: %w", err)
	}

	if err := installation.Schema().Validate(toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s", path)
	}

	return base, nil
}

// CreateV1Alpha1Lock writes the resolved casting to the lock file.
func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, machinery v1alpha1.Machinery, path string) error {
	contents, err := domain.MarshalYAML(machinery)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"), contents, 0644); err != nil {
		return fmt.Errorf("failed to write yaml file: %w", err)
	}

	return nil
}

// GetV1Alpha1Lock reads the lock file and dispatches by kind.
func (*yamlConfig) GetV1Alpha1Lock(ctx context.Context, path string) (v1alpha1.Machinery, error) {
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"))
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml file: %w", err)
	}

	kind, err := peekKind(bytes)
	if err != nil {
		return nil, err
	}

	switch kind {
	case v1alpha1.KindInstallation:
		var c installation.Casting
		if err := domain.UnmarshalYAML(bytes, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal installation casting: %w", err)
		}
		return &c, nil
	}
	return nil, errors.Newf(errors.TypeUnsupported, "unknown casting kind %q", kind)
}
