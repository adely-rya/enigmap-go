package avl

type NodeID uint64

const NullAddress NodeID = 0

func selectNodeID(current, candidate NodeID, condition int) NodeID {
	mask := uint64(0) - uint64(condition&1)
	return NodeID(
		(uint64(current) &^ mask) |
			(uint64(candidate) & mask),
	)
}

func nodeIDPresent(id NodeID) int {
	value := uint64(id)
	return int((value | (0 - value)) >> 63)
}

type Node[V any] struct {
	Key   Key // key component of ptr in the paper
	Value V   // v in the paper

	Height int

	Left  NodeID // address component of lptr in the paper
	Right NodeID // address component of rptr in the paper
	Valid uint8
}
