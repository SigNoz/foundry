// Package convention derives the names and tags a provisioned substrate is
// identified by.
//
// Foundry generates rather than reconciles, so it can never ask a platform what
// it created. A consuming casting finds a producing casting's resources only by
// deriving the same names and filtering the same tags. Both sides are derived
// here so neither can drift.
//
// Provider limits are absent: a length cap belongs to the platform enforcing it,
// so a casting measures what it derives against its own provider's limits.
package convention

import (
	"regexp"

	"github.com/signoz/foundry/internal/errors"
)

// namePattern is what the strictest provider accepts as a name segment, and is
// shared by every name a caller supplies.
var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// maxNameLength matches the metadata.name cap in the casting schema.
const maxNameLength = 63

// Substrate is the infrastructure an installation runs on, known by the
// provisioning casting's metadata.name. Every resource is named and tagged from
// it, and it is the only fact a consumer needs to find them.
type Substrate struct {
	name string
}

func NewSubstrate(name string) (Substrate, error) {
	if name == "" {
		return Substrate{}, errors.Newf(errors.TypeInvalidInput, "failed to create substrate from %q: name is empty", name)
	}

	if len(name) > maxNameLength {
		return Substrate{}, errors.Newf(errors.TypeInvalidInput, "failed to create substrate from %q: name is longer than %d characters", name, maxNameLength)
	}

	if !namePattern.MatchString(name) {
		return Substrate{}, errors.Newf(errors.TypeInvalidInput, "failed to create substrate from %q: name is not lowercase alphanumeric with interior hyphens", name)
	}

	return Substrate{name: name}, nil
}

func MustNewSubstrate(name string) Substrate {
	substrate, err := NewSubstrate(name)
	if err != nil {
		panic(err)
	}

	return substrate
}

func (s Substrate) String() string {
	return s.name
}
