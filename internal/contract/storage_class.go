package contract

import "github.com/signoz/foundry/internal/errors"

var (
	// StorageClassPersistent nodes each carry a volume that outlives them.
	// Stateful identities claim those volumes, which is what pins the group:
	// a node cannot be swapped for another without moving a claim.
	StorageClassPersistent = StorageClass{s: "persistent", data: true, pinned: true}

	// StorageClassEphemeral nodes are interchangeable and keep nothing.
	StorageClassEphemeral = StorageClass{s: "ephemeral"}
)

// StorageClass is the durability of a node group's storage. Each class carries
// what it implies, so a new class is one var entry rather than another branch
// wherever groups are checked.
type StorageClass struct {
	s      string
	data   bool
	pinned bool
}

func ParseStorageClass(value string) (StorageClass, error) {
	for _, class := range StorageClasses() {
		if class.String() == value {
			return class, nil
		}
	}

	return StorageClass{}, errors.Newf(errors.TypeInvalidInput, "failed to create storage class from %q: it names no class", value)
}

func (class StorageClass) String() string {
	return class.s
}

// RequiresDataVolume reports whether nodes of this class must declare a volume
// that outlives them, and conversely that other classes must not.
func (class StorageClass) RequiresDataVolume() bool {
	return class.data
}

// IsPinned reports whether the group's size is fixed. Every node in a pinned
// group owns a claimed volume, so there is nothing to scale between bounds.
func (class StorageClass) IsPinned() bool {
	return class.pinned
}

func StorageClasses() []StorageClass {
	return []StorageClass{StorageClassPersistent, StorageClassEphemeral}
}
