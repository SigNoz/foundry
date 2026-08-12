package contract

import "github.com/signoz/foundry/internal/errors"

var (
	// StorageClassPersistent nodes each carry a volume that outlives them and is
	// claimed by an identity, so the group is pinned.
	StorageClassPersistent = StorageClass{s: "persistent", data: true, pinned: true}

	// StorageClassEphemeral nodes are interchangeable and keep nothing.
	StorageClassEphemeral = StorageClass{s: "ephemeral"}
)

// StorageClass is the durability of a node group's storage, and carries what
// that implies for the group's volumes and bounds.
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
// that outlives them. Other classes must not.
func (class StorageClass) RequiresDataVolume() bool {
	return class.data
}

// IsPinned reports whether the group's size is fixed, meaning minSize and
// maxSize must be equal.
func (class StorageClass) IsPinned() bool {
	return class.pinned
}

func StorageClasses() []StorageClass {
	return []StorageClass{StorageClassPersistent, StorageClassEphemeral}
}
