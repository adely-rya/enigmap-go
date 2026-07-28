package avl

import "math/bits"

type Tree[V any] struct {
	root     NodeID
	fake     NodeID
	maxDepth int
	store    NodeStore[V]
}

func NewTree[V any](store NodeStore[V], maxNodes uint64) (*Tree[V], error) {
	fakeID, err := store.Allocate(Node[V]{Valid: 0})
	if err != nil {
		return nil, err
	}

	fakeNode := Node[V]{
		Left:  fakeID,
		Right: fakeID,
		Valid: 0,
	}
	if err := store.Update(fakeID, fakeNode); err != nil {
		return nil, err
	}

	return &Tree[V]{
		root:     NullAddress,
		fake:     fakeID,
		maxDepth: calculateMaxDepth(maxNodes),
		store:    store,
	}, nil
}

func calculateMaxDepth(maxNodes uint64) int {
	// This is the public ENIGMAP implementation's safe approximation of
	// the AVL bound of 1.44*log2(N). The extra two slots are reserved for
	// the fake node and the root sentinel.
	baseDepth := bits.Len64(maxNodes + 1)
	return baseDepth + baseDepth/2 + 1
}

func (t *Tree[V]) Root() NodeID {
	return t.root
}
