package memory

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestStoreAllocateLoadAndUpdate(t *testing.T) {
	store := New[string]()
	key := avl.NewKey(make([]byte, avl.KeySize))

	id, err := store.Allocate(avl.Node[string]{Key: key, Value: "first"})
	if err != nil {
		t.Fatalf("allocate node: %v", err)
	}
	if id == avl.NullAddress {
		t.Fatal("allocated null address")
	}

	node, err := store.Load(id)
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	if node.Value != "first" {
		t.Fatalf("unexpected value: %q", node.Value)
	}

	node.Value = "updated"
	if err := store.Update(id, node); err != nil {
		t.Fatalf("update node: %v", err)
	}

	node, err = store.Load(id)
	if err != nil {
		t.Fatalf("load updated node: %v", err)
	}
	if node.Value != "updated" {
		t.Fatalf("unexpected updated value: %q", node.Value)
	}
}

func TestStoreRejectsUnknownNode(t *testing.T) {
	store := New[string]()

	if _, err := store.Load(42); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("load error = %v, want %v", err, ErrNodeNotFound)
	}
	if err := store.Update(42, avl.Node[string]{}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("update error = %v, want %v", err, ErrNodeNotFound)
	}
	if err := store.FreeMaybe(42, 1); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("free error = %v, want %v", err, ErrNodeNotFound)
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
	if _, err := store.Load(id); err != nil {
		t.Fatalf("candidate was freed with condition=0: %v", err)
	}

	if err := store.FreeMaybe(id, 1); err != nil {
		t.Fatalf("FreeMaybe(condition=1): %v", err)
	}
	if _, err := store.Load(id); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("load freed node error = %v, want %v", err, ErrNodeNotFound)
	}

	if err := store.FreeMaybe(42, 0); err != nil {
		t.Fatalf("FreeMaybe(unknown, condition=0): %v", err)
	}
}
