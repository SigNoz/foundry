package yamlconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

type yamlConfig struct {
	loaders map[v1alpha1.Kind]loaderFn
}

type loaderFn func(bytes []byte, path string) (v1alpha1.Machinery, error)

func New() config.Config {
	return &yamlConfig{
		loaders: map[v1alpha1.Kind]loaderFn{
			v1alpha1.KindInstallation:    loadInstallation,
			v1alpha1.KindCollectionAgent: loadCollectionAgent,
		},
	}
}

// GetV1Alpha1 reads, peeks at kind, dispatches to the per-Kind loader, merges
// defaults, validates against the per-Kind schema, and returns the resolved
// casting wrapped as v1alpha1.Machinery.
func (c *yamlConfig) GetV1Alpha1(ctx context.Context, path string) (v1alpha1.Machinery, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read yaml file")
	}

	kind, err := peekKind(bytes)
	if err != nil {
		return nil, err
	}

	load, ok := c.loaders[kind]
	if !ok {
		return nil, errors.Newf(errors.TypeUnsupported, "unknown casting kind %q", kind)
	}
	return load(bytes, path)
}

// peekKind decodes only the kind field from raw bytes. Empty or missing kind
// defaults to KindInstallation so existing castings without `kind` keep working.
func peekKind(bytes []byte) (v1alpha1.Kind, error) {
	var probe struct {
		Kind v1alpha1.Kind `json:"kind" yaml:"kind"`
	}
	if err := domain.UnmarshalYAML(bytes, &probe); err != nil {
		return v1alpha1.Kind{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to peek kind")
	}
	if probe.Kind == (v1alpha1.Kind{}) {
		return v1alpha1.KindInstallation, nil
	}
	return probe.Kind, nil
}

func loadInstallation(bytes []byte, path string) (v1alpha1.Machinery, error) {
	var loaded installation.Casting
	if err := domain.UnmarshalYAML(bytes, &loaded); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal installation casting")
	}

	base := installation.Default()
	if err := v1alpha1.Merge(base, &loaded); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to merge default installation casting")
	}

	contents, err := json.Marshal(base)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to marshal installation casting")
	}
	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal installation casting for validation")
	}

	if err := installation.Schema().Validate(toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s", path)
	}

	return base, nil
}

func loadCollectionAgent(bytes []byte, path string) (v1alpha1.Machinery, error) {
	var loaded collectionagent.Casting
	if err := domain.UnmarshalYAML(bytes, &loaded); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal collectionagent casting")
	}

	base := collectionagent.Default()
	if err := v1alpha1.Merge(base, &loaded); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to merge default collectionagent casting")
	}

	contents, err := json.Marshal(base)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to marshal collectionagent casting")
	}
	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal collectionagent casting for validation")
	}

	if err := collectionagent.Schema().Validate(toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s", path)
	}

	return base, nil
}

// CreateV1Alpha1Lock writes the resolved casting to the lock file.
func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, machinery v1alpha1.Machinery, path string) error {
	contents, err := domain.MarshalYAML(machinery)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to marshal yaml")
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"), contents, 0644); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to write yaml file")
	}

	return nil
}

// GetV1Alpha1Lock reads the lock file and dispatches by kind.
func (*yamlConfig) GetV1Alpha1Lock(ctx context.Context, path string) (v1alpha1.Machinery, error) {
	bytes, err := os.ReadFile(filepath.Join(filepath.Dir(path), "casting.yaml.lock"))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read yaml file")
	}

	kind, err := peekKind(bytes)
	if err != nil {
		return nil, err
	}

	switch kind {
	case v1alpha1.KindInstallation:
		var c installation.Casting
		if err := domain.UnmarshalYAML(bytes, &c); err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal installation casting")
		}
		return &c, nil
	case v1alpha1.KindCollectionAgent:
		var c collectionagent.Casting
		if err := domain.UnmarshalYAML(bytes, &c); err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal collectionagent casting")
		}
		return &c, nil
	}
	return nil, errors.Newf(errors.TypeUnsupported, "unknown casting kind %q", kind)
}
