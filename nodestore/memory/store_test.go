package memory

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestStoreAllocateLoadAndUpdate(t *testing.T) {
	store := New[string]()

	id, err := store.Allocate(avl.Node[string]{Key: []byte("key"), Value: "first"})
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
}
