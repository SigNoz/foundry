package contract

// Ownership is whether the substrate created a resource or adopted an existing
// one. The zero value is owned.
type Ownership struct {
	s      string
	shared bool
}

var (
	OwnershipOwned  = Ownership{s: "owned"}
	OwnershipShared = Ownership{s: "shared", shared: true}
)

func (ownership Ownership) String() string {
	if ownership.s == "" {
		return OwnershipOwned.s
	}

	return ownership.s
}

func (ownership Ownership) IsShared() bool {
	return ownership.shared
}
