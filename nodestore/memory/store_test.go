package memory

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestStoreAllocateAndAccess(t *testing.T) {
	store := New[string]()
	key := avl.NewKey(make([]byte, avl.KeySize))

	id, err := store.Allocate(avl.Node[string]{Key: key, Value: "first"})
	if err != nil {
		t.Fatalf("allocate node: %v", err)
	}
	if id.Address == avl.NullAddress {
		t.Fatal("allocated null address")
	}

	node, err := store.BeginAccess(id, store.NewPosition())
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	if node.Value != "first" {
		t.Fatalf("unexpected value: %q", node.Value)
	}

	node.Value = "updated"
	if err := store.FinishAccess(id.Address, node); err != nil {
		t.Fatalf("update node: %v", err)
	}

	node, err = store.BeginAccess(id, store.NewPosition())
	if err != nil {
		t.Fatalf("load updated node: %v", err)
	}
	if node.Value != "updated" {
		t.Fatalf("unexpected updated value: %q", node.Value)
	}
	if err := store.FinishAccess(id.Address, node); err != nil {
		t.Fatalf("finish updated node access: %v", err)
	}
}

func TestStoreRejectsUnknownNode(t *testing.T) {
	store := New[string]()

	unknown := avl.Pointer{Address: 42}
	if _, err := store.BeginAccess(unknown, store.NewPosition()); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("load error = %v, want %v", err, ErrNodeNotFound)
	}
	if err := store.FinishAccess(42, avl.Node[string]{}); !errors.Is(err, ErrNoAccessInProgress) {
		t.Fatalf("finish error = %v, want %v", err, ErrNoAccessInProgress)
	}
	if err := store.FreeMaybe(unknown, 1); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("free error = %v, want %v", err, ErrNodeNotFound)
	}
}

func TestStoreEnforcesAccessLifecycle(t *testing.T) {
	store := New[string]()
	first, err := store.Allocate(avl.Node[string]{Value: "first"})
	if err != nil {
		t.Fatalf("allocate first node: %v", err)
	}
	second, err := store.Allocate(avl.Node[string]{Value: "second"})
	if err != nil {
		t.Fatalf("allocate second node: %v", err)
	}

	node, err := store.BeginAccess(first, store.NewPosition())
	if err != nil {
		t.Fatalf("begin first access: %v", err)
	}
	if _, err := store.BeginAccess(second, store.NewPosition()); !errors.Is(err, ErrAccessInProgress) {
		t.Fatalf("nested begin error = %v, want %v", err, ErrAccessInProgress)
	}
	if err := store.FinishAccess(second.Address, node); !errors.Is(err, ErrAccessIDMismatch) {
		t.Fatalf("mismatched finish error = %v, want %v", err, ErrAccessIDMismatch)
	}
	if err := store.FinishAccess(first.Address, node); err != nil {
		t.Fatalf("finish first access: %v", err)
	}
	if err := store.FinishAccess(first.Address, node); !errors.Is(err, ErrNoAccessInProgress) {
		t.Fatalf("duplicate finish error = %v, want %v", err, ErrNoAccessInProgress)
	}
}

func TestStoreFreeMaybe(t *testing.T) {
	store := New[string]()
	id, err := store.Allocate(avl.Node[string]{Value: "candidate"})
	if err != nil {
		t.Fatalf("allocate node: %v", err)
	}

	if err := store.FreeMaybe(id, 0); err != nil {
		t.Fatalf("FreeMaybe(condition=0): %v", err)
	}
	node, err := store.BeginAccess(id, store.NewPosition())
	if err != nil {
		t.Fatalf("candidate was freed with condition=0: %v", err)
	}
	if err := store.FinishAccess(id.Address, node); err != nil {
		t.Fatalf("finish candidate access: %v", err)
	}

	if err := store.FreeMaybe(id, 1); err != nil {
		t.Fatalf("FreeMaybe(condition=1): %v", err)
	}
	if _, err := store.BeginAccess(id, store.NewPosition()); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("load freed node error = %v, want %v", err, ErrNodeNotFound)
	}

	if err := store.FreeMaybe(avl.Pointer{Address: 42}, 0); err != nil {
		t.Fatalf("FreeMaybe(unknown, condition=0): %v", err)
	}
}
