package avl

type NodeID uint64

const NullAddress NodeID = 0

type Node[V any] struct {
	Key   Key // key component of ptr in the paper
	Value V   // v in the paper

	Height int

	Left  NodeID // address component of lptr in the paper
	Right NodeID // address component of rptr in the paper
	Valid uint8
}
