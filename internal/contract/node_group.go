package contract

// NodeGroup is a pool of interchangeable nodes, named by its key and selected
// by its storage class.
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
