package domain

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/signoz/foundry/internal/errors"
)

// Version is a parsed semantic version. It keeps the original string (including
// any build flavor such as "-alpine" or "-distroless") so it renders losslessly,
// but compares on the core major.minor.patch, so flavor, build metadata, and
// ClickHouse's 4-part tags do not affect compatibility decisions.
type Version struct {
	raw string
	sv  *semver.Version
}

// ParseVersion parses a version tag such as "25.12.5", "25.12.5-alpine", or
// ClickHouse's "25.12.5.44". A non-semver tag like "latest" returns an error;
// callers treat that as an unknown version.
func ParseVersion(raw string) (Version, error) {
	sv, err := semver.NewVersion(raw)
	if err != nil {
		// ClickHouse publishes 4-part tags (e.g. "25.12.5.44") that semver
		// rejects; fall back to the leading major.minor.patch core.
		core, ok := coreVersion(raw)
		if !ok {
			return Version{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to parse version from %q: not a semantic version", raw)
		}
		sv = core
	}

	return Version{raw: raw, sv: sv}, nil
}

// MustParseVersion is ParseVersion for known-good literals; it panics on error.
func MustParseVersion(raw string) Version {
	version, err := ParseVersion(raw)
	if err != nil {
		panic(err)
	}

	return version
}

// Satisfies reports whether the version's core major.minor.patch meets the
// constraints. Flavor and pre-release identifiers are ignored, so
// "25.12.5-alpine" satisfies ">=25.12.5".
func (v Version) Satisfies(constraints *semver.Constraints) bool {
	return constraints.Check(semver.New(v.sv.Major(), v.sv.Minor(), v.sv.Patch(), "", ""))
}

// String renders the original version string, flavor included.
func (v Version) String() string {
	return v.raw
}

// coreVersion extracts a leading "major.minor.patch" from a tag that semver
// rejects, dropping any flavor suffix and extra version segments.
func coreVersion(raw string) (*semver.Version, bool) {
	base := strings.TrimPrefix(raw, "v")
	if i := strings.IndexByte(base, '-'); i >= 0 {
		base = base[:i]
	}

	parts := strings.SplitN(base, ".", 4)
	if len(parts) < 3 {
		return nil, false
	}

	sv, err := semver.NewVersion(parts[0] + "." + parts[1] + "." + parts[2])
	if err != nil {
		return nil, false
	}

	return sv, true
}
