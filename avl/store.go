package avl

type NodeStore[V any] interface {
	Allocate(node Node[V]) (Pointer, error)
	NewPosition() Position
	BeginAccess(pointer Pointer, newPosition Position) (Node[V], error)
	FinishAccess(address NodeID, node Node[V]) error
	FreeMaybe(pointer Pointer, condition int) error
}
