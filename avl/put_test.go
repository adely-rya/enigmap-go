package avl

import (
	"errors"
	"testing"
)

func TestPutAllocatesCandidateNode(t *testing.T) {
	store := newTreeTestStore[string]()
	tree, err := NewTree[string](store, 8)
	if err != nil {
		t.Fatalf("NewTree(): %v", err)
	}
	allocatedBefore := len(store.allocated)
	uncacheBefore := store.uncacheCount
	store.loadIDs = nil
	store.updatedIDs = nil
	key := getTestKey(0x20)

	if err := tree.Put(key, "candidate"); err != nil {
		t.Fatalf("Put(): %v", err)
	}

	if len(store.allocated) != allocatedBefore+1 {
		t.Fatalf(
			"Allocate() count increased by %d, want 1",
			len(store.allocated)-allocatedBefore,
		)
	}
	candidate := store.allocated[len(store.allocated)-1]
	if candidate.Key != NewKey(key) {
		t.Fatal("candidate Key does not match Put key")
	}
	if candidate.Value != "candidate" {
		t.Fatalf("candidate Value = %q, want %q", candidate.Value, "candidate")
	}
	if candidate.Balance != BalanceBalanced {
		t.Fatalf(
			"candidate Balance = %d, want %d",
			candidate.Balance,
			BalanceBalanced,
		)
	}
	if candidate.Left != tree.fake || candidate.Right != tree.fake {
		t.Fatalf(
			"candidate children = (%d, %d), want (%d, %d)",
			candidate.Left,
			candidate.Right,
			tree.fake,
			tree.fake,
		)
	}
	if candidate.Valid != 1 {
		t.Fatalf("candidate Valid = %d, want 1", candidate.Valid)
	}
	if len(store.loadIDs) != 2*tree.maxDepth {
		t.Fatalf(
			"BeginAccess count = %d, want %d",
			len(store.loadIDs),
			2*tree.maxDepth,
		)
	}
	assertPairedAccesses(t, store.loadIDs, store.updatedIDs)
	if store.uncacheCount != uncacheBefore+1 {
		t.Fatalf(
			"UncacheAll count increased by %d, want 1",
			store.uncacheCount-uncacheBefore,
		)
	}
}

func TestPutPropagatesCandidateAllocationError(t *testing.T) {
	store := newTreeTestStore[string]()
	tree, err := NewTree[string](store, 8)
	if err != nil {
		t.Fatalf("NewTree(): %v", err)
	}
	store.allocateErr = errStoreOperation

	if err := tree.Put(getTestKey(0x20), "candidate"); !errors.Is(err, errStoreOperation) {
		t.Fatalf("Put() error = %v, want %v", err, errStoreOperation)
	}
}
