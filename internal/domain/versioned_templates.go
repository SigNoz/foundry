package domain

import "github.com/signoz/foundry/internal/errors"

// VersionedTemplates dispatches to a Template by a component version string. It
// is the selection primitive for materials whose content can diverge across
// component versions (for example, per-ClickHouse-version server configs). When
// a requested version has no dedicated template, Resolve falls back to the
// latest supported version, so pinning an unknown version still produces output.
type VersionedTemplates struct {
	latest    string
	templates map[string]*Template
}

// NewVersionedTemplates builds a dispatch table keyed by version. latest is the
// fallback version returned for unknown lookups and must have a template.
func NewVersionedTemplates(latest string, templates map[string]*Template) (*VersionedTemplates, error) {
	if _, ok := templates[latest]; !ok {
		return nil, errors.Newf(errors.TypeInvalidInput, "failed to create versioned templates: latest version %q has no template", latest)
	}
	return &VersionedTemplates{latest: latest, templates: templates}, nil
}

// MustNewVersionedTemplates is like NewVersionedTemplates but panics on error.
// Use for package-level tables built from known-good literals.
func MustNewVersionedTemplates(latest string, templates map[string]*Template) *VersionedTemplates {
	vt, err := NewVersionedTemplates(latest, templates)
	if err != nil {
		panic(err)
	}
	return vt
}

// Resolve returns the template for version, or the latest-version template when
// version has no dedicated entry. The bool reports whether an exact match was
// found, letting callers warn on fallback.
func (vt *VersionedTemplates) Resolve(version string) (*Template, bool) {
	if t, ok := vt.templates[version]; ok {
		return t, true
	}
	return vt.templates[vt.latest], false
}
