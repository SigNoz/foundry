// Package contract derives the names and tags a provisioned substrate is
// identified by, so that the casting which provisions it and the casting which
// consumes it arrive at the same values without reading the platform.
//
// Provider name-length caps are not enforced here.
package contract

import (
	"regexp"

	"github.com/signoz/foundry/internal/errors"
)

// namePattern is what the strictest provider accepts as a name segment.
var namePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// maxNameLength matches the metadata.name cap in the casting schema.
const maxNameLength = 63

// Substrate is the infrastructure an installation runs on, named by the
// provisioning casting's metadata.name. Every derived name and tag comes from
// that name alone.
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
