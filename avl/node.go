package avl

type NodeID uint64

const NullAddress NodeID = 0

type Node[V any] struct {
	Key   []byte
	Value V

	Height int

	Left  NodeID
	Right NodeID
}
