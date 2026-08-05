package convention

import (
	"slices"
	"strconv"
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

const identitySeparator = ","

// Identity is a stateful seat that claims a volume and keeps its data across an
// instance replacement: a component and its ordinals, "telemetrystore-0-0". It
// carries no substrate prefix, because deployed claims are spelled this way.
type Identity struct {
	s string
}

func NewIdentity(component string, ordinals ...int) (Identity, error) {
	if component == "" {
		return Identity{}, errors.Newf(errors.TypeInvalidInput, "failed to create identity: component is empty")
	}

	// The separator carries the encoding, so one inside a component would split
	// into two identities on the way back.
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

// ParseIdentity reads back a claimed identity. Trailing numeric segments are the
// ordinals; the rest is the component, which may itself be hyphenated.
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

// Identities is the claim record one volume carries, encoded as a single tag
// value: sorted and separator-joined, matching Terraform's join and split.
// Sorting keeps the value stable so an unchanged claim set produces no diff.
//
// Only a platform with no stateful identity primitive of its own needs a claim
// record. Kubernetes binds a pod to its volume through the StatefulSet
// controller, compose and swarm by name in the generated file, systemd by host
// path. Empty is therefore the norm, and stamps no tag.
type Identities []Identity

func (identities Identities) String() string {
	parts := make([]string, 0, len(identities))
	for _, identity := range identities.sorted() {
		parts = append(parts, identity.s)
	}

	return strings.Join(parts, identitySeparator)
}

// ParseIdentities is the counterpart of String, validating through ParseIdentity
// so there is one path into the type.
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
