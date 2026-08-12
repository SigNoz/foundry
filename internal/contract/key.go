package contract

import (
	"github.com/signoz/foundry/internal/errors"
)

// Key is the operator-chosen reference for one declared subnet or instance
// group, lowercase alphanumeric with interior hyphens. It is the qualifier in
// every name derived for that thing.
type Key struct {
	s string
}

func NewKey(key string) (Key, error) {
	if key == "" {
		return Key{}, errors.Newf(errors.TypeInvalidInput, "failed to create key from %q: key is empty", key)
	}

	if !namePattern.MatchString(key) {
		return Key{}, errors.Newf(errors.TypeInvalidInput, "failed to create key from %q: key is not lowercase alphanumeric with interior hyphens", key)
	}

	return Key{s: key}, nil
}

func MustNewKey(key string) Key {
	parsed, err := NewKey(key)
	if err != nil {
		panic(err)
	}

	return parsed
}

func (key Key) String() string {
	return key.s
}
