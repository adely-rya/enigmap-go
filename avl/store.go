package avl

type AccessToken struct {
	Address NodeID
}

type NodeStore[V any] interface {
	NewPosition() (ORAMPosition, error)
	BeginAccess(pointer Pointer, newPosition ORAMPosition) (Node[V], AccessToken, error)
	FinishAccess(token AccessToken, node Node[V]) error
	Allocate(node Node[V]) (Pointer, error)
	FreeMaybe(pointer Pointer, condition int) error
	UncacheAll()
}
