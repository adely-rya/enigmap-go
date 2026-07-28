package avl

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/omap"
)

var errStoreOperation = errors.New("store operation failed")

type treeTestStore[V any] struct {
	nextID       NodeID
	nodes        map[NodeID]Node[V]
	loadIDs      []NodeID
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

func (s *treeTestStore[V]) Allocate(node Node[V]) (NodeID, error) {
	if s.allocateErr != nil {
		return 0, s.allocateErr
	}

	id := s.nextID
	s.nextID++
	s.nodes[id] = node
	s.allocated = append(s.allocated, node)
	return id, nil
}

func (s *treeTestStore[V]) Load(id NodeID) (Node[V], error) {
	s.loadIDs = append(s.loadIDs, id)
	node, ok := s.nodes[id]
	if !ok {
		return Node[V]{}, errStoreOperation
	}
	return node, nil
}

func (s *treeTestStore[V]) Update(id NodeID, node Node[V]) error {
	if s.updateErr != nil {
		return s.updateErr
	}

	s.nodes[id] = node
	s.updatedIDs = append(s.updatedIDs, id)
	s.updatedNodes = append(s.updatedNodes, node)
	return nil
}

func (s *treeTestStore[V]) FreeMaybe(id NodeID, condition int) error {
	if condition&1 == 0 {
		return nil
	}
	if _, ok := s.nodes[id]; !ok {
		return errStoreOperation
	}
	delete(s.nodes, id)
	s.freedIDs = append(s.freedIDs, id)
	return nil
}

var stringSelector = omap.ValueSelectorFunc[string](
	func(current, candidate string, condition int) string {
		if condition == 1 {
			return candidate
		}
		return current
	},
)

func TestNewTreeCreatesFakeNodeAndRootSentinel(t *testing.T) {
	store := newTreeTestStore[string]()

	tree, err := NewTree[string](store, 1024, stringSelector)
	if err != nil {
		t.Fatalf("NewTree(): %v", err)
	}

	if tree.fake != 7 {
		t.Fatalf("fake node ID = %d, want 7", tree.fake)
	}
	fake, err := store.Load(tree.fake)
	if err != nil {
		t.Fatalf("load fake node: %v", err)
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

	if tree.root != 8 {
		t.Fatalf("root sentinel ID = %d, want 8", tree.root)
	}
	root, err := store.Load(tree.root)
	if err != nil {
		t.Fatalf("load root sentinel: %v", err)
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

		if _, err := NewTree[string](store, 1, stringSelector); !errors.Is(err, errStoreOperation) {
			t.Fatalf("NewTree() error = %v, want %v", err, errStoreOperation)
		}
	})

	t.Run("update", func(t *testing.T) {
		store := newTreeTestStore[string]()
		store.updateErr = errStoreOperation

		if _, err := NewTree[string](store, 1, stringSelector); !errors.Is(err, errStoreOperation) {
			t.Fatalf("NewTree() error = %v, want %v", err, errStoreOperation)
		}
	})

	t.Run("nil selector", func(t *testing.T) {
		store := newTreeTestStore[string]()

		if _, err := NewTree[string](store, 1, nil); !errors.Is(err, ErrNilValueSelector) {
			t.Fatalf("NewTree() error = %v, want %v", err, ErrNilValueSelector)
		}
	})
}
