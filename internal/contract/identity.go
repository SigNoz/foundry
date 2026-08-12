package contract

import (
	"slices"
	"strconv"
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

const identitySeparator = ","

// Identity is a component and its ordinals, "telemetrystore-0-0". It claims a
// volume, which is what keeps the component's data across a node replacement.
type Identity struct {
	s string
}

func NewIdentity(component string, ordinals ...int) (Identity, error) {
	if component == "" {
		return Identity{}, errors.Newf(errors.TypeInvalidInput, "failed to create identity: component is empty")
	}

	// A separator inside a component would split into two on the way back.
	if strings.Contains(component, identitySeparator) {
		return Identity{}, errors.Newf(errors.TypeInvalidInput, "failed to create identity from %q: component contains %q", component, identitySeparator)
	}

	parts := make([]string, 0, len(ordinals)+1)
	parts = append(parts, component)

	for _, ordinal := range ordinals {
		if ordinal < 0 {
			return Identity{}, errors.Newf(errors.TypeInvalidInput, "failed to create identity from %q: ordinal %d is negative", component, ordinal)
		}

		parts = append(parts, strconv.Itoa(ordinal))
	}

	return Identity{s: strings.Join(parts, "-")}, nil
}

func MustNewIdentity(component string, ordinals ...int) Identity {
	identity, err := NewIdentity(component, ordinals...)
	if err != nil {
		panic(err)
	}

	return identity
}

// ParseIdentity splits a claimed identity: trailing numeric segments are the
// ordinals, the rest is the component, which may itself be hyphenated.
func ParseIdentity(value string) (Identity, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return Identity{}, errors.Newf(errors.TypeInvalidInput, "failed to create identity from %q: identity is empty", value)
	}

	segments := strings.Split(trimmed, "-")

	ordinals := make([]int, 0, len(segments))
	boundary := len(segments)

	for boundary > 1 {
		ordinal, err := strconv.Atoi(segments[boundary-1])
		if err != nil {
			break
		}

		ordinals = append([]int{ordinal}, ordinals...)
		boundary--
	}

	return NewIdentity(strings.Join(segments[:boundary], "-"), ordinals...)
}

func (identity Identity) String() string {
	return identity.s
}

// Identities is the set of claims one volume carries, comma-separated and
// sorted so that an unchanged set produces no diff. GCP labels reject the
// comma.
type Identities []Identity

func (identities Identities) String() string {
	parts := make([]string, 0, len(identities))
	for _, identity := range identities.sorted() {
		parts = append(parts, identity.s)
	}

	return strings.Join(parts, identitySeparator)
}

func ParseIdentities(value string) (Identities, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, identitySeparator)

	identities := make(Identities, 0, len(parts))
	for _, part := range parts {
		identity, err := ParseIdentity(part)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create identities from %q", value)
		}

		identities = append(identities, identity)
	}

	return identities.sorted(), nil
}

func (identities Identities) sorted() Identities {
	out := slices.Clone(identities)
	slices.SortFunc(out, func(a, b Identity) int { return strings.Compare(a.s, b.s) })

	return out
}
