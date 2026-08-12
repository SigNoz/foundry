package contract

import ()

// NodeGroup is a pool of interchangeable nodes. The key names it. The storage
// class selects it, being the only fact about it a consumer can predict.
type NodeGroup struct {
	key     Key
	storage StorageClass
}

func NewNodeGroup(key Key, storage StorageClass) NodeGroup {
	return NodeGroup{key: key, storage: storage}
}

func (group NodeGroup) Key() Key {
	return group.key
}

func (group NodeGroup) Storage() StorageClass {
	return group.storage
}
