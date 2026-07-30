package avl

import (
	"errors"
	"testing"
)

var errStoreOperation = errors.New("store operation failed")

type treeTestStore[V any] struct {
	nextID       NodeID
	nodes        map[NodeID]Node[V]
	loadIDs      []Pointer
	activeID     NodeID
	allocateErr  error
	updateErr    error
	allocated    []Node[V]
	updatedIDs   []NodeID
	updatedNodes []Node[V]
	freedIDs     []NodeID
}

func newTreeTestStore[V any]() *treeTestStore[V] {
	return &treeTestStore[V]{
		nextID: 7,
		nodes:  make(map[NodeID]Node[V]),
	}
}

func (s *treeTestStore[V]) Allocate(node Node[V]) (Pointer, error) {
	if s.allocateErr != nil {
		return NullPointer, s.allocateErr
	}

	id := s.nextID
	s.nextID++
	s.nodes[id] = node
	s.allocated = append(s.allocated, node)
	return Pointer{Address: id}, nil
}

func (s *treeTestStore[V]) NewPosition() Position {
	return 0
}

func (s *treeTestStore[V]) BeginAccess(
	pointer Pointer,
	newPosition Position,
) (Node[V], error) {
	if s.activeID != NullAddress {
		return Node[V]{}, errStoreOperation
	}
	s.loadIDs = append(s.loadIDs, pointer)
	node, ok := s.nodes[pointer.Address]
	if !ok {
		return Node[V]{}, errStoreOperation
	}
	s.activeID = pointer.Address
	return node, nil
}

func (s *treeTestStore[V]) FinishAccess(id NodeID, node Node[V]) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if s.activeID != id {
		return errStoreOperation
	}

	s.nodes[id] = node
	s.activeID = NullAddress
	s.updatedIDs = append(s.updatedIDs, id)
	s.updatedNodes = append(s.updatedNodes, node)
	return nil
}

func (s *treeTestStore[V]) FreeMaybe(pointer Pointer, condition int) error {
	if condition&1 == 0 {
		return nil
	}
	if _, ok := s.nodes[pointer.Address]; !ok {
		return errStoreOperation
	}
	delete(s.nodes, pointer.Address)
	s.freedIDs = append(s.freedIDs, pointer.Address)
	return nil
}

func TestNewTreeCreatesFakeNodeAndRootSentinel(t *testing.T) {
	store := newTreeTestStore[string]()

	tree, err := NewTree[string](store, 1024)
	if err != nil {
		t.Fatalf("NewTree(): %v", err)
	}

	if tree.fake.Address != 7 {
		t.Fatalf("fake node ID = %d, want 7", tree.fake.Address)
	}
	fake, err := store.BeginAccess(tree.fake, store.NewPosition())
	if err != nil {
		t.Fatalf("load fake node: %v", err)
	}
	if err := store.FinishAccess(tree.fake.Address, fake); err != nil {
		t.Fatalf("finish fake node access: %v", err)
	}
	if fake.Left != tree.fake || fake.Right != tree.fake {
		t.Fatalf(
			"fake children = (%d, %d), want (%d, %d)",
			fake.Left,
			fake.Right,
			tree.fake,
			tree.fake,
		)
	}
	if fake.Valid != 0 {
		t.Fatalf("fake Valid = %d, want 0", fake.Valid)
	}
	if fake.Balance != BalanceBalanced {
		t.Fatalf("fake Balance = %d, want %d", fake.Balance, BalanceBalanced)
	}

	if tree.root.Address != 8 {
		t.Fatalf("root sentinel ID = %d, want 8", tree.root.Address)
	}
	root, err := store.BeginAccess(tree.root, store.NewPosition())
	if err != nil {
		t.Fatalf("load root sentinel: %v", err)
	}
	if err := store.FinishAccess(tree.root.Address, root); err != nil {
		t.Fatalf("finish root sentinel access: %v", err)
	}
	if root.Left != tree.fake || root.Right != tree.fake {
		t.Fatalf(
			"root sentinel children = (%d, %d), want (%d, %d)",
			root.Left,
			root.Right,
			tree.fake,
			tree.fake,
		)
	}
	if root.Valid != 0 {
		t.Fatalf("root sentinel Valid = %d, want 0", root.Valid)
	}
	if root.Balance != BalanceBalanced {
		t.Fatalf("root sentinel Balance = %d, want %d", root.Balance, BalanceBalanced)
	}
}

func TestNewTreeCalculatesPublicImplementationDepthBound(t *testing.T) {
	tests := []struct {
		maxNodes uint64
		want     int
	}{
		{maxNodes: 0, want: 2},
		{maxNodes: 1, want: 4},
		{maxNodes: 2, want: 4},
		{maxNodes: 1024, want: 17},
	}

	for _, tt := range tests {
		if got := calculateMaxDepth(tt.maxNodes); got != tt.want {
			t.Fatalf(
				"calculateMaxDepth(%d) = %d, want %d",
				tt.maxNodes,
				got,
				tt.want,
			)
		}
	}
}

func TestNewTreePropagatesStoreErrors(t *testing.T) {
	t.Run("allocate", func(t *testing.T) {
		store := newTreeTestStore[string]()
		store.allocateErr = errStoreOperation

		if _, err := NewTree[string](store, 1); !errors.Is(err, errStoreOperation) {
			t.Fatalf("NewTree() error = %v, want %v", err, errStoreOperation)
		}
	})

	t.Run("update", func(t *testing.T) {
		store := newTreeTestStore[string]()
		store.updateErr = errStoreOperation

		if _, err := NewTree[string](store, 1); !errors.Is(err, errStoreOperation) {
			t.Fatalf("NewTree() error = %v, want %v", err, errStoreOperation)
		}
	})
}
