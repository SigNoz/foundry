package convention

import (
	"github.com/signoz/foundry/internal/errors"
)

// Key is the reference an operator chooses for one declared thing, a subnet or
// an instance group. Identity lives in the key. A partial override restates the
// key and nothing else.
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
