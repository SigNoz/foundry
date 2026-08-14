package yamlconfig

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

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

var _ config.Config = (*yamlConfig)(nil)

const lockFileName = "casting.yaml.lock"

type yamlConfig struct {
	loaders map[v1alpha1.Kind]loader
}

// loader is one Kind's entry in the table. Supporting a new Kind is one entry
// here and its place in v1alpha1.Kinds().
type loader struct {
	kind v1alpha1.Kind

	new func() v1alpha1.Machinery

	// defaults returns the Kind's baseline. It reads the declaration because a
	// Kind may default one component from another's declared kind.
	defaults func(declared v1alpha1.Machinery) v1alpha1.Machinery

	schema func() *jsonschema.Resolved

	// check is the Kind's compatibility gate; nil when it has none.
	check func(casting v1alpha1.Machinery) error

	// discriminator tells the Kind's documents apart; nil for a Kind that
	// holds one document per file.
	discriminator *discriminator
}

// discriminator tells two documents of a Kind apart, so a casting file may
// hold one document per value; name is the property errors call it by.
type discriminator struct {
	name string
	of   func(casting v1alpha1.Machinery) string
}

// reader turns one document into its casting: loader.resolve for the casting
// file, loader.read for the lock.
type reader func(loader, []byte) (v1alpha1.Machinery, error)

// resolve lays the Kind's defaults under the declaration, merges the
// declaration over them, validates the result against the Kind's schema, and
// runs the Kind's compatibility check.
func (l loader) resolve(document []byte) (v1alpha1.Machinery, error) {
	declared := l.new()
	if err := domain.UnmarshalYAML(document, declared); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal %s casting", l.kind)
	}

	casting := l.defaults(declared)
	if err := v1alpha1.Merge(casting, declared); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to merge %s casting over its defaults", l.kind)
	}

	// The schema describes the JSON shape, so the casting validates as JSON.
	contents, err := json.Marshal(casting)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to marshal %s casting", l.kind)
	}

	toValidate := map[string]any{}
	if err := json.Unmarshal(contents, &toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal %s casting for validation", l.kind)
	}

	if err := l.schema().Validate(toValidate); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to validate %s casting against its schema", l.kind)
	}

	if l.check != nil {
		if err := l.check(casting); err != nil {
			return nil, err
		}
	}

	return casting, nil
}

// read unmarshals an already-resolved casting; the lock's documents take
// neither defaults nor validation.
func (l loader) read(document []byte) (v1alpha1.Machinery, error) {
	casting := l.new()
	if err := domain.UnmarshalYAML(document, casting); err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal %s casting", l.kind)
	}

	return casting, nil
}

// collide rejects a casting whose pours would overwrite an already-declared
// one's: a second document of a Kind without a discriminator, or one with the
// same discriminator value. Values are compared after defaults are merged, so
// an omitted value collides with an explicitly declared default.
func (l loader) collide(casting v1alpha1.Machinery, declared []v1alpha1.Machinery) error {
	for _, existing := range declared {
		if l.discriminator == nil {
			return errors.Newf(errors.TypeInvalidInput, "%s is already declared as %q: a casting file holds one %s", l.kind, existing.Name(), l.kind)
		}

		if value := l.discriminator.of(casting); value == l.discriminator.of(existing) {
			return errors.Newf(errors.TypeInvalidInput, "%s is already declared as %q: a casting file holds one %s per %s and both declare %q", l.kind, existing.Name(), l.kind, l.discriminator.name, value)
		}
	}

	return nil
}

func New(logger *slog.Logger) *yamlConfig {
	loaders := []loader{
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
			discriminator: &discriminator{
				name: "collector kind",
				of: func(casting v1alpha1.Machinery) string {
					return casting.(*collectionagent.Casting).Spec.Collector.Kind.String()
				},
			},
		},
	}

	byKind := make(map[v1alpha1.Kind]loader, len(loaders))
	for _, l := range loaders {
		byKind[l.kind] = l
	}

	return &yamlConfig{loaders: byKind}
}

// GetV1Alpha1 reads, dispatches, and validates every casting document the file
// holds, and returns the resolved castings in cast order.
func (config *yamlConfig) GetV1Alpha1(ctx context.Context, path string) ([]v1alpha1.Machinery, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read casting file")
	}

	return config.castings(contents, path, loader.resolve)
}

// GetV1Alpha1Lock reads the lock beside the casting file; its documents are
// already resolved, so they take neither defaults nor validation.
func (config *yamlConfig) GetV1Alpha1Lock(ctx context.Context, path string) ([]v1alpha1.Machinery, error) {
	lockPath := filepath.Join(filepath.Dir(path), lockFileName)

	contents, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to read lock file")
	}

	return config.castings(contents, lockPath, loader.read)
}

// CreateV1Alpha1Lock writes the resolved castings beside the casting file, one
// document each, in the order they were resolved.
func (*yamlConfig) CreateV1Alpha1Lock(ctx context.Context, machineries []v1alpha1.Machinery, path string) error {
	documents := make([][]byte, 0, len(machineries))

	for _, machinery := range machineries {
		contents, err := domain.MarshalYAML(machinery)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to marshal %s casting %q", machinery.Kind(), machinery.Name())
		}

		documents = append(documents, contents)
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(path), lockFileName), domain.NewYAMLStream(documents), 0644); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to write lock file")
	}

	return nil
}

// castings reads every casting document in the stream and returns them in cast
// order, a Kind's own documents in file order. Positions in errors count every
// document in the file, empty ones included.
func (config *yamlConfig) castings(contents []byte, path string, read reader) ([]v1alpha1.Machinery, error) {
	byKind := make(map[v1alpha1.Kind][]v1alpha1.Machinery, len(config.loaders))
	count := 0

	for position, document := range domain.YAMLStream(contents).Documents() {
		kind, err := peekKind(document)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s: document %d", path, position)
		}

		loader, supported := config.loaders[kind]
		if !supported {
			return nil, errors.Newf(errors.TypeUnsupported, "invalid casting file %s: document %d: no loader for casting kind %q", path, position, kind)
		}

		casting, err := read(loader, document)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s: document %d", path, position)
		}

		if err := loader.collide(casting, byKind[kind]); err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid casting file %s: document %d", path, position)
		}

		byKind[kind] = append(byKind[kind], casting)
		count++
	}

	if count == 0 {
		return nil, errors.Newf(errors.TypeInvalidInput, "invalid casting file %s: it declares no castings", path)
	}

	// v1alpha1.Kinds() is cast order.
	machineries := make([]v1alpha1.Machinery, 0, count)
	for _, kind := range v1alpha1.Kinds() {
		machineries = append(machineries, byKind[kind]...)
	}

	return machineries, nil
}

// peekKind reads a document's kind alone. A missing kind is an Installation,
// so casting files written before kind was introduced keep working.
func peekKind(document []byte) (v1alpha1.Kind, error) {
	var probe struct {
		Kind v1alpha1.Kind `json:"kind" yaml:"kind"`
	}

	if err := domain.UnmarshalYAML(document, &probe); err != nil {
		return v1alpha1.Kind{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to read kind")
	}

	if probe.Kind == (v1alpha1.Kind{}) {
		return v1alpha1.KindInstallation, nil
	}

	return probe.Kind, nil
}
