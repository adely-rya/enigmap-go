package avl

import "math/bits"

// maxAVLDepth is the largest value calculateMaxDepth can produce for a
// uint64 node bound. It lets Put keep its public-size planning buffers on the
// goroutine stack instead of allocating slices on every call.
const maxAVLDepth = 64 + 64/2 + 1

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
	defer store.UncacheAll()

	fakeID, err := store.Allocate(Node[V]{
		Balance: BalanceBalanced,
		Valid:   0,
	})
	if err != nil {
		return nil, err
	}
	fakeOld := fakeID
	fakePosition, err := store.NewPosition()
	if err != nil {
		return nil, err
	}
	fake := pointerWithPosition(fakeOld, fakePosition)

	fakeNode := Node[V]{
		Balance: BalanceBalanced,
		Left:    fake,
		Right:   fake,
		Valid:   0,
	}
	_, token, err := store.BeginAccess(fakeOld, fakePosition)
	if err != nil {
		return nil, err
	}
	if err := store.FinishAccess(token, fakeNode); err != nil {
		return nil, err
	}

	root, err := store.Allocate(Node[V]{
		Balance: BalanceBalanced,
		Left:    fake,
		Right:   fake,
		Valid:   0,
	})
	if err != nil {
		return nil, err
	}

	return &Tree[V]{
		root:     root,
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
