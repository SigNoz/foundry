// Package mechanic implements the foundryctl mechanic verbs (status, diagnose,
// inspect) that diagnose a running deployment. It owns the resource-path
// grammar shared across those verbs and, in time, the catalog of checks
// partitioned by casting Kind and molding.
package mechanic

import (
	"slices"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/errors"
)

// Resource is a parsed mechanic resource path.
type Resource struct {
	Kind         v1alpha1.Kind
	KindExplicit bool
	Molding      v1alpha1.MoldingKind
	EntityKind   string
	EntityID     string
}

// Overrides carries optional connection details supplied via flags or environment, used when the lock file is unavailable.
type Overrides struct {
	Signoz        string
	ClickhouseDSN string
	MetastoreDSN  string
}

var moldingsByKind = map[v1alpha1.Kind][]v1alpha1.MoldingKind{
	v1alpha1.KindInstallation: {
		v1alpha1.MoldingKindSignoz,
		v1alpha1.MoldingKindTelemetryStore,
		v1alpha1.MoldingKindMetaStore,
		v1alpha1.MoldingKindIngester,
		v1alpha1.MoldingKindTelemetryKeeper,
	},
	v1alpha1.KindCollectionAgent: {
		v1alpha1.MoldingKindCollector,
	},
}

// ParseResource normalizes a resource path into a Resource.
func ParseResource(args []string) (Resource, error) {
	raw := strings.Join(args, "/")

	segments := make([]string, 0, len(args))
	for s := range strings.SplitSeq(raw, "/") {
		if s = strings.TrimSpace(s); s != "" {
			segments = append(segments, s)
		}
	}

	if len(segments) == 0 {
		return Resource{}, errors.Newf(errors.TypeInvalidInput, "resource path is empty")
	}

	var res Resource

	if kind, ok := matchKind(segments[0]); ok {
		res.Kind = kind
		res.KindExplicit = true
		segments = segments[1:]
	}

	if len(segments) == 0 {
		return Resource{}, errors.Newf(errors.TypeInvalidInput, "resource path %q is missing a molding", raw)
	}

	if err := res.Molding.UnmarshalText([]byte(segments[0])); err != nil {
		return Resource{}, errors.Wrapf(err, errors.TypeInvalidInput, "resource path %q", raw)
	}
	segments = segments[1:]

	if len(segments) > 0 {
		res.EntityKind = segments[0]
		segments = segments[1:]
	}
	if len(segments) > 0 {
		res.EntityID = segments[0]
		segments = segments[1:]
	}
	if len(segments) > 0 {
		return Resource{}, errors.Newf(errors.TypeInvalidInput, "resource path %q has too many segments", raw)
	}

	return res, nil
}

// Resolve fills an implicit Kind from the casting's kind and verifies the molding belongs to that kind.
func (r Resource) Resolve(kind v1alpha1.Kind) (Resource, error) {
	if !r.KindExplicit {
		r.Kind = kind
	}

	moldings, ok := moldingsByKind[r.Kind]
	if !ok {
		return Resource{}, errors.Newf(errors.TypeUnsupported, "unsupported casting kind %q", r.Kind)
	}

	if slices.Contains(moldings, r.Molding) {
		return r, nil
	}

	return Resource{}, errors.Newf(errors.TypeInvalidInput, "molding %q is not valid for kind %q", r.Molding, r.Kind)
}

// String renders the resource in canonical slash form.
func (r Resource) String() string {
	parts := make([]string, 0, 4)
	if r.Kind.String() != "" {
		parts = append(parts, r.Kind.String())
	}
	parts = append(parts, r.Molding.String())
	if r.EntityKind != "" {
		parts = append(parts, r.EntityKind)
	}
	if r.EntityID != "" {
		parts = append(parts, r.EntityID)
	}
	return strings.Join(parts, "/")
}

func matchKind(s string) (v1alpha1.Kind, bool) {
	for _, kind := range v1alpha1.Kinds() {
		if strings.EqualFold(kind.String(), s) {
			return kind, true
		}
	}
	return v1alpha1.Kind{}, false
}
