package convention

import (
	"github.com/signoz/foundry/api/v1alpha1"
)

// NodeGroup is a pool of interchangeable nodes. The key names it. The storage
// class selects it, being the only fact about it a consumer can predict.
type NodeGroup struct {
	key     Key
	storage v1alpha1.StorageClass
}

func NewNodeGroup(key Key, storage v1alpha1.StorageClass) NodeGroup {
	return NodeGroup{key: key, storage: storage}
}

func (group NodeGroup) Key() Key {
	return group.key
}

func (group NodeGroup) Storage() v1alpha1.StorageClass {
	return group.storage
}
