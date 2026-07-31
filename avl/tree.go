package avl

import "math/bits"

type Tree[V any] struct {
	root     Pointer
	fake     Pointer
	maxDepth int
	store    NodeStore[V]
}

func NewTree[V any](
	store NodeStore[V],
	maxNodes uint64,
) (*Tree[V], error) {
	fakeID, err := store.Allocate(Node[V]{
		Balance: BalanceBalanced,
		Valid:   0,
	})
	if err != nil {
		return nil, err
	}
	fake := Pointer{Address: fakeID}

	fakeNode := Node[V]{
		Balance: BalanceBalanced,
		Left:    fake,
		Right:   fake,
		Valid:   0,
	}
	if err := store.Update(fakeID, fakeNode); err != nil {
		return nil, err
	}

	rootID, err := store.Allocate(Node[V]{
		Balance: BalanceBalanced,
		Left:    fake,
		Right:   fake,
		Valid:   0,
	})
	if err != nil {
		return nil, err
	}

	return &Tree[V]{
		root:     Pointer{Address: rootID},
		fake:     fake,
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

// AVL探索は、終了した深さを隠すため常にmaxDepth回のnode accessを行う。

// Root returns the fixed root sentinel ID.
func (t *Tree[V]) Root() Pointer {
	return t.root
}
