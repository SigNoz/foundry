package contract

import (
	"slices"
	"strconv"
	"strings"

	"github.com/signoz/foundry/internal/errors"
)

const identitySeparator = ","

// Identity is a stateful seat that claims a volume and keeps its data across an
// instance replacement: a component and its ordinals, "telemetrystore-0-0".
// Deployed claims carry no substrate prefix.
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

// Identities is the claim record one volume carries, sorted and
// separator-joined to match Terraform's join and split. Sorting keeps an
// unchanged claim set from producing a diff.
//
// Only platforms without a stateful identity primitive need this. Kubernetes,
// compose, swarm and systemd bind an identity to its disk themselves. The comma
// encoding is legal on AWS and Azure, illegal on GCP.
type Identities []Identity

func (identities Identities) String() string {
	parts := make([]string, 0, len(identities))
	for _, identity := range identities.sorted() {
		parts = append(parts, identity.s)
	}

	return strings.Join(parts, identitySeparator)
}

// ParseIdentities is the counterpart of String. It validates through
// ParseIdentity, keeping one path into the type.
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
