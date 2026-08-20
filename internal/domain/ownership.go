package domain

import (
	"maps"
	"slices"
	"strings"
)

// Owner is whoever a workload belongs to, as the attributes a platform records
// on it: labels on a container, tags on a cloud resource. Two workloads share
// an owner when every attribute they were asked for matches, so an owner is
// compared as a whole and never by one attribute at a time.
type Owner map[string]string

// IsZero reports an owner that recorded nothing. A workload carrying none of
// the attributes asked for belongs to no one foundry can name, which is not
// the same as belonging to an owner whose every attribute is empty.
func (owner Owner) IsZero() bool {
	for _, value := range owner {
		if value != "" {
			return false
		}
	}

	return true
}

// Equal treats an absent attribute and an empty one as the same, so an owner
// asked for fewer attributes still compares against one asked for more.
func (owner Owner) Equal(other Owner) bool {
	for key, value := range owner {
		if other[key] != value {
			return false
		}
	}

	for key, value := range other {
		if owner[key] != value {
			return false
		}
	}

	return true
}

// String renders the attributes in key order, so the same owner always reads
// the same way in a message.
func (owner Owner) String() string {
	pairs := make([]string, 0, len(owner))
	for _, key := range slices.Sorted(maps.Keys(owner)) {
		pairs = append(pairs, key+"="+owner[key])
	}

	return strings.Join(pairs, ",")
}

// Ownership is the owners a group of workloads reports, one owner per
// workload, deduplicated.
type Ownership struct {
	owners  []Owner
	unowned bool
}

// NewOwnership records what each workload reported. A workload that recorded
// nothing marks the group as partly unowned rather than becoming an owner in
// its own right.
func NewOwnership(owners ...Owner) Ownership {
	ownership := Ownership{}

	for _, owner := range owners {
		if owner.IsZero() {
			ownership.unowned = true
			continue
		}

		if slices.ContainsFunc(ownership.owners, owner.Equal) {
			continue
		}

		ownership.owners = append(ownership.owners, owner)
	}

	return ownership
}

// Foreign returns an owner that is not self, when one exists.
func (ownership Ownership) Foreign(self Owner) (Owner, bool) {
	for _, owner := range ownership.owners {
		if !owner.Equal(self) {
			return owner, true
		}
	}

	return nil, false
}

// HasUnowned reports workloads that recorded no owner: either a deployment
// made before foundry stamped them, or a foreign one sharing the same name.
func (ownership Ownership) HasUnowned() bool {
	return ownership.unowned
}
