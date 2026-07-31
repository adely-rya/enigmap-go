package avl

import "testing"

func getTestKey(first byte) []byte {
	key := make([]byte, KeySize)
	key[0] = first
	return key
}

func getTestNode(first byte, value string, fake Pointer) Node[string] {
	return Node[string]{
		Key:   NewKey(getTestKey(first)),
		Value: value,
		Left:  fake,
		Right: fake,
		Valid: 1,
	}
}

func TestGetAlwaysLoadsMaxDepth(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		store := newTreeTestStore[string]()
		tree, err := NewTree[string](store, 8)
		if err != nil {
			t.Fatalf("NewTree(): %v", err)
		}

		store.loadIDs = nil
		store.updatedIDs = nil
		_, found, err := tree.Get(getTestKey(0x20))
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if found {
			t.Fatal("Get() found key in empty tree")
		}
		assertFixedLoads(t, store.loadIDs, tree.maxDepth)
		assertPairedAccesses(t, store.loadIDs, store.updatedIDs)
		if store.loadIDs[0] != tree.root.Address {
			t.Fatalf("first Load() = %d, want root sentinel %d", store.loadIDs[0], tree.root.Address)
		}
		assertLoadsFrom(t, store.loadIDs, 1, tree.fake.Address)
	})

	t.Run("found at root", func(t *testing.T) {
		store := newTreeTestStore[string]()
		tree, err := NewTree[string](store, 8)
		if err != nil {
			t.Fatalf("NewTree(): %v", err)
		}

		root, err := store.Allocate(getTestNode(0x20, "root", tree.fake))
		if err != nil {
			t.Fatalf("allocate root: %v", err)
		}
		setTestRoot(t, tree, store, root)

		store.loadIDs = nil
		store.updatedIDs = nil
		value, found, err := tree.Get(getTestKey(0x20))
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if !found || value != "root" {
			t.Fatalf("Get() = (%q, %v), want (%q, true)", value, found, "root")
		}
		assertFixedLoads(t, store.loadIDs, tree.maxDepth)
		assertPairedAccesses(t, store.loadIDs, store.updatedIDs)
		if store.loadIDs[0] != tree.root.Address || store.loadIDs[1] != root.Address {
			t.Fatalf(
				"first Loads = %v, want [%d %d]",
				store.loadIDs[:2],
				tree.root.Address,
				root.Address,
			)
		}
		assertLoadsFrom(t, store.loadIDs, 2, tree.fake.Address)
	})

	t.Run("found below root", func(t *testing.T) {
		store := newTreeTestStore[string]()
		tree, err := NewTree[string](store, 8)
		if err != nil {
			t.Fatalf("NewTree(): %v", err)
		}

		left, err := store.Allocate(getTestNode(0x10, "left", tree.fake))
		if err != nil {
			t.Fatalf("allocate left: %v", err)
		}
		rootNode := getTestNode(0x20, "root", tree.fake)
		rootNode.Left = left
		root, err := store.Allocate(rootNode)
		if err != nil {
			t.Fatalf("allocate root: %v", err)
		}
		setTestRoot(t, tree, store, root)

		store.loadIDs = nil
		store.updatedIDs = nil
		value, found, err := tree.Get(getTestKey(0x10))
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if !found || value != "left" {
			t.Fatalf("Get() = (%q, %v), want (%q, true)", value, found, "left")
		}
		assertFixedLoads(t, store.loadIDs, tree.maxDepth)
		assertPairedAccesses(t, store.loadIDs, store.updatedIDs)
		if store.loadIDs[0] != tree.root.Address ||
			store.loadIDs[1] != root.Address ||
			store.loadIDs[2] != left.Address {
			t.Fatalf(
				"first Loads = %v, want [%d %d %d]",
				store.loadIDs[:3],
				tree.root.Address,
				root.Address,
				left.Address,
			)
		}
		assertLoadsFrom(t, store.loadIDs, 3, tree.fake.Address)
	})

	t.Run("missing", func(t *testing.T) {
		store := newTreeTestStore[string]()
		tree, err := NewTree[string](store, 8)
		if err != nil {
			t.Fatalf("NewTree(): %v", err)
		}

		root, err := store.Allocate(getTestNode(0x20, "root", tree.fake))
		if err != nil {
			t.Fatalf("allocate root: %v", err)
		}
		setTestRoot(t, tree, store, root)

		store.loadIDs = nil
		store.updatedIDs = nil
		_, found, err := tree.Get(getTestKey(0x30))
		if err != nil {
			t.Fatalf("Get(): %v", err)
		}
		if found {
			t.Fatal("Get() found missing key")
		}
		assertFixedLoads(t, store.loadIDs, tree.maxDepth)
		assertPairedAccesses(t, store.loadIDs, store.updatedIDs)
		if store.loadIDs[0] != tree.root.Address || store.loadIDs[1] != root.Address {
			t.Fatalf(
				"first Loads = %v, want [%d %d]",
				store.loadIDs[:2],
				tree.root.Address,
				root.Address,
			)
		}
		assertLoadsFrom(t, store.loadIDs, 2, tree.fake.Address)
	})
}

func setTestRoot(
	t *testing.T,
	tree *Tree[string],
	store *treeTestStore[string],
	root Pointer,
) {
	t.Helper()
	sentinel, err := store.Load(tree.root.Address)
	if err != nil {
		t.Fatalf("load root sentinel: %v", err)
	}
	sentinel.Left = root
	if err := store.Update(tree.root.Address, sentinel); err != nil {
		t.Fatalf("update root sentinel: %v", err)
	}
}

func assertFixedLoads(t *testing.T, loads []NodeID, want int) {
	t.Helper()
	if len(loads) != want {
		t.Fatalf("Load() count = %d, want %d: %v", len(loads), want, loads)
	}
}

func assertLoadsFrom(t *testing.T, loads []NodeID, start int, want NodeID) {
	t.Helper()
	for i := start; i < len(loads); i++ {
		if loads[i] != want {
			t.Fatalf("Load()[%d] = %d, want fake %d", i, loads[i], want)
		}
	}
}

func assertPairedAccesses(t *testing.T, begun, finished []NodeID) {
	t.Helper()
	if len(begun) != len(finished) {
		t.Fatalf(
			"BeginAccess count = %d, FinishAccess count = %d",
			len(begun),
			len(finished),
		)
	}
	for index := range begun {
		if begun[index] != finished[index] {
			t.Fatalf(
				"access[%d]: BeginAccess address = %d, FinishAccess address = %d",
				index,
				begun[index],
				finished[index],
			)
		}
	}
}
