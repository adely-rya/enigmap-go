package avl

type NodeID uint64

const NullAddress NodeID = 0

type Node[V any] struct {
	Key   Key
	Value V

	Height int

	Left  NodeID
	Right NodeID
}
