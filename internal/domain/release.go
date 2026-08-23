package domain

import "github.com/signoz/foundry/internal/errors"

// Release is foundry's reading of one deployment: constructed, never stored.
type Release struct {
	// Name is the unit's identity (metadata.name).
	Name string

	// Owner is what the unit asserts on the host.
	Owner Owner
}

func (r Release) Validate() error {
	if r.Name == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no name is stated")
	}

	if len(r.Owner) == 0 {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no owner is stated")
	}

	return nil
}
