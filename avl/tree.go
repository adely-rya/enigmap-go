package avl

type Tree[V any] struct {
	root  NodeID
	store NodeStore[V]
}

func NewTree[V any](store NodeStore[V]) *Tree[V] {
	return &Tree[V]{
		root:  NullAddress,
		store: store,
	}
}

func (t *Tree[V]) Root() NodeID {
	return t.root
}
