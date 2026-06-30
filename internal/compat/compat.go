// Package compat evaluates version-compatibility rules between deployment
// components. It is Kind-agnostic: it operates only on molding kinds and
// resolved versions, so any casting Kind can supply its own rule set.
package compat

import (
	"log/slog"

	"github.com/Masterminds/semver/v3"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

// Rule declares a compatibility requirement: when the Subject component's
// version satisfies When, the Target component's version must satisfy Requires.
// When and Requires are semver constraint strings, so a rule is a plain literal.
// Subject and Target are typed molding kinds so a rule cannot name a component
// that does not exist.
type Rule struct {
	Subject  v1alpha1.MoldingKind
	When     string
	Target   v1alpha1.MoldingKind
	Requires string
	Advice   string
}

// Resolved is a component's resolved version. It is built by NewResolved.
type Resolved struct {
	tag     string
	version domain.Version
	known   bool
	present bool
}

// NewResolved resolves a component's version from its image. present reports
// whether the component is enabled; a non-semver tag (such as "latest") yields
// a present-but-unknown version.
func NewResolved(image string, present bool) Resolved {
	resolved := Resolved{present: present}
	if ref, err := domain.ParseImage(image); err == nil {
		resolved.tag = ref.Tag()
		if version, ok := ref.Version(); ok {
			resolved.version = version
			resolved.known = true
		}
	}

	return resolved
}

// Check evaluates rules against the resolved component versions. A provably
// incompatible concrete pairing is a hard error; a floating subject tag that
// could become incompatible is a warning.
func Check(versions map[v1alpha1.MoldingKind]Resolved, rules []Rule, logger *slog.Logger) error {
	for _, rule := range rules {
		when, err := semver.NewConstraint(rule.When)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "invalid 'when' constraint %q for %s", rule.When, rule.Subject)
		}

		requires, err := semver.NewConstraint(rule.Requires)
		if err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "invalid 'requires' constraint %q for %s", rule.Requires, rule.Target)
		}

		subject := versions[rule.Subject]
		target := versions[rule.Target]
		if !subject.present || !target.present {
			continue // a disabled or absent component cannot conflict
		}

		if !target.known || target.version.Satisfies(requires) {
			continue // target version is unknown, or already meets the requirement
		}

		switch {
		case subject.known && subject.version.Satisfies(when):
			return errors.Newf(errors.TypeInvalidInput,
				"%s %s is incompatible with %s %s: %s",
				rule.Subject, subject.tag, rule.Target, target.tag, rule.Advice)
		case !subject.known:
			logger.Warn("floating component tag may become incompatible with a dependency",
				slog.String("component", rule.Subject.String()),
				slog.String("tag", subject.tag),
				slog.String("advice", rule.Advice))
		}
	}

	return nil
}
