package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/jsonschema-go"
	"go.yaml.in/yaml/v3"
)

var _ yaml.Marshaler = (*StorageClass)(nil)
var _ yaml.Unmarshaler = (*StorageClass)(nil)
var _ json.Marshaler = (*StorageClass)(nil)
var _ json.Unmarshaler = (*StorageClass)(nil)
var _ fmt.Stringer = (*StorageClass)(nil)
var _ jsonschema.Enum = (*StorageClass)(nil)

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

func (class StorageClass) MarshalJSON() ([]byte, error) {
	return json.Marshal(class.String())
}

func (class *StorageClass) UnmarshalJSON(text []byte) error {
	var str string
	if err := json.Unmarshal(text, &str); err != nil {
		return err
	}

	return class.UnmarshalText([]byte(str))
}

func (class *StorageClass) UnmarshalText(text []byte) error {
	for _, availableClass := range StorageClasses() {
		if availableClass.String() == string(text) {
			*class = availableClass
			return nil
		}
	}

	// A nil slice is an absent value, which leaves the zero class; an empty
	// string is a stated value that names no class, and falls through.
	if text == nil {
		*class = StorageClass{}
		return nil
	}

	return errors.New("invalid storage class: " + string(text))
}

func (class StorageClass) MarshalText() ([]byte, error) {
	return []byte(class.String()), nil
}

func (class *StorageClass) UnmarshalYAML(node *yaml.Node) error {
	return class.UnmarshalText([]byte(node.Value))
}

func (class StorageClass) MarshalYAML() (any, error) {
	return class.String(), nil
}

func (class StorageClass) Enum() []any {
	classes := []any{}
	for _, class := range StorageClasses() {
		classes = append(classes, class.String())
	}

	return classes
}
