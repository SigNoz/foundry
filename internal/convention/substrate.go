// Package convention derives the names and tags a provisioned substrate is
// identified by. Foundry never reads back from a platform, so a producing and a
// consuming casting must work them out the same way. Both are derived here.
//
// Provider limits are absent. A casting measures what it derives against its
// own provider's caps.
package convention

import (
	"regexp"

	"github.com/signoz/foundry/internal/errors"
)

// namePattern is what the strictest provider accepts as a name segment.
var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// maxNameLength matches the metadata.name cap in the casting schema.
const maxNameLength = 63

// Substrate is the infrastructure an installation runs on, known by the
// provisioning casting's metadata.name. A consumer needs no other fact to find
// every resource.
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
