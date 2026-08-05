package convention

// Ownership is whether the substrate created a resource or adopted one it must
// never delete.
type Ownership struct {
	s      string
	shared bool
}

var (
	OwnershipOwned  = Ownership{s: "owned"}
	OwnershipShared = Ownership{s: "shared", shared: true}
)

// String resolves the zero value to owned, which is what saying nothing means.
func (ownership Ownership) String() string {
	if ownership.s == "" {
		return OwnershipOwned.s
	}

	return ownership.s
}

func (ownership Ownership) IsShared() bool {
	return ownership.shared
}
