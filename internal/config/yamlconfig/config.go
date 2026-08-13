package yamlconfig

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/api/v1alpha1/infrastructure"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	installationcompat "github.com/signoz/foundry/internal/compat/installation"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

const lockFileName = "casting.yaml.lock"

// loader is one Kind's entry in the table. Supporting a new Kind is one entry
// in yamlConfig.loaders and nothing else.
type loader struct {
	kind v1alpha1.Kind

	// new returns an empty casting of the Kind.
	new func() v1alpha1.Machinery

	// defaults returns the Kind's baseline. It reads the declaration because a
	// Kind may default one component from another's declared kind.
	defaults func(declared v1alpha1.Machinery) v1alpha1.Machinery

	// schema is the Kind's published shape.
	schema func() *jsonschema.Resolved

	// check is the Kind's compatibility matrix. Nil when it has none.
	check func(casting v1alpha1.Machinery) error
}

// resolve lays the Kind's defaults under the declaration, merges the
// declaration over them, and validates the result against the Kind's schema.
func (l loader) resolve(bytes []byte) (v1alpha1.Machinery, error) {
	declared := l.new()
	if err := domain.UnmarshalYAML(bytes, declared); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal %s casting", l.kind)
	}

	casting := l.defaults(declared)
	if err := v1alpha1.Merge(casting, declared); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to merge default %s casting", l.kind)
	}

	if err := validate(casting, l.schema()); err != nil {
		return nil, err
	}

	if l.check == nil {
		return casting, nil
	}

	if err := l.check(casting); err != nil {
		return nil, err
	}

	return casting, nil
}

// read unmarshals an already-resolved casting. The lock's documents take
// neither defaults nor validation.
func (l loader) read(bytes []byte) (v1alpha1.Machinery, error) {
	casting := l.new()
	if err := domain.UnmarshalYAML(bytes, casting); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal %s casting", l.kind)
	}

	return casting, nil
}

type yamlConfig struct {
	// loaders is one entry per Kind, in cast order: a substrate before the
	// workloads that run on it.
	loaders []loader
}

func New(logger *slog.Logger) config.Config {
	return &yamlConfig{
		loaders: []loader{
			{
				kind:     v1alpha1.KindInfrastructure,
				new:      func() v1alpha1.Machinery { return &infrastructure.Casting{} },
				defaults: func(v1alpha1.Machinery) v1alpha1.Machinery { return infrastructure.Default() },
				schema:   infrastructure.Schema,
			},
			{
				kind: v1alpha1.KindInstallation,
				new:  func() v1alpha1.Machinery { return &installation.Casting{} },
				defaults: func(declared v1alpha1.Machinery) v1alpha1.Machinery {
					return installation.Default(declared.(*installation.Casting))
				},
				schema: installation.Schema,
				check: func(casting v1alpha1.Machinery) error {
					return installationcompat.Compatibility(casting.(*installation.Casting), logger)
				},
			},
			{
				kind:     v1alpha1.KindCollectionAgent,
				new:      func() v1alpha1.Machinery { return &collectionagent.Casting{} },
				defaults: func(v1alpha1.Machinery) v1alpha1.Machinery { return collectionagent.Default() },
				schema:   collectionagent.Schema,
			},
		},
	}
}

// GetV1Alpha1 reads, dispatches, and validates every casting document in the
// file, and returns the resolved castings in cast order.
func (c *yamlConfig) GetV1Alpha1(ctx context.Context, path string) ([]v1alpha1.Machinery, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read yaml file")
	}

	return c.castings(contents, path, loader.resolve)
}

// GetV1Alpha1Lock reads the lock file, whose documents are already resolved:
// no defaults, no validation.
func (c *yamlConfig) GetV1Alpha1Lock(ctx context.Context, path string) ([]v1alpha1.Machinery, error) {
	lockPath := filepath.Join(filepath.Dir(path), lockFileName)

	contents, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read yaml file")
	}

	return c.castings(contents, lockPath, loader.read)
}

// CreateV1Alpha1Lock writes the resolved castings to the lock file, one
// document each, in the order they were resolved.
func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, machineries []v1alpha1.Machinery, path string) error {
	documents := make([][]byte, 0, len(machineries))
	for _, machinery := range machineries {
		contents, err := domain.MarshalYAML(machinery)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to marshal yaml")
		}
		documents = append(documents, contents)
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(path), lockFileName), domain.NewYAMLStream(documents), 0644); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to write yaml file")
	}

	return nil
}

// castings reads every casting document in the stream with its Kind's reader
// and returns them in cast order. Documents that declare nothing are skipped.
// Positions in errors count every document in the file, as a reader does.
func (c *yamlConfig) castings(contents []byte, path string, read func(loader, []byte) (v1alpha1.Machinery, error)) ([]v1alpha1.Machinery, error) {
	byKind := make(map[v1alpha1.Kind]v1alpha1.Machinery, len(c.loaders))

	for position, bytes := range domain.YAMLStream(contents).Documents() {
		var probe struct {
			Kind v1alpha1.Kind `json:"kind" yaml:"kind"`
		}
		if err := domain.UnmarshalYAML(bytes, &probe); err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s: document %d", path, position)
		}

		if probe.Kind == (v1alpha1.Kind{}) {
			return nil, errors.Newf(errors.TypeInvalidInput, "invalid casting file %s: document %d: kind is required", path, position)
		}

		// A Kind appears at most once: two documents of a Kind would pour into
		// the same directory.
		if declared, taken := byKind[probe.Kind]; taken {
			return nil, errors.Newf(errors.TypeInvalidInput, "invalid casting file %s: document %d: %s is declared twice, already as %q: a casting file holds at most one document of a kind", path, position, probe.Kind, declared.Name())
		}

		index := slices.IndexFunc(c.loaders, func(l loader) bool { return l.kind == probe.Kind })
		if index < 0 {
			return nil, errors.Newf(errors.TypeUnsupported, "invalid casting file %s: document %d: unknown casting kind %q", path, position, probe.Kind)
		}

		machinery, err := read(c.loaders[index], bytes)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s: document %d (%s)", path, position, probe.Kind)
		}

		byKind[probe.Kind] = machinery
	}

	if len(byKind) == 0 {
		return nil, errors.Newf(errors.TypeInvalidInput, "invalid casting file %s: it declares no castings", path)
	}

	// Walking the loaders emits the set in cast order.
	machineries := make([]v1alpha1.Machinery, 0, len(byKind))
	for _, l := range c.loaders {
		if machinery, declared := byKind[l.kind]; declared {
			machineries = append(machineries, machinery)
		}
	}

	return machineries, nil
}

// validate checks the casting against its schema through JSON, which is the
// shape the schema describes.
func validate(casting v1alpha1.Machinery, schema *jsonschema.Resolved) error {
	contents, err := json.Marshal(casting)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to marshal casting")
	}

	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal casting for validation")
	}

	if err := schema.Validate(toValidate); err != nil {
		return errors.Wrapf(err, errors.TypeInvalidInput, "failed to validate casting against its schema")
	}

	return nil
}
