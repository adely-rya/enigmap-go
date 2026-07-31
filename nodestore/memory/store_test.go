package memory

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestStoreAllocateBeginAndFinishAccess(t *testing.T) {
	store := New[string]()
	key := avl.NewKey(make([]byte, avl.KeySize))

	pointer, err := store.Allocate(avl.Node[string]{Key: key, Value: "first"})
	if err != nil {
		t.Fatalf("allocate node: %v", err)
	}
	if pointer.Address == avl.NullAddress {
		t.Fatal("allocated null address")
	}
	if pointer.Position != 0 {
		t.Fatalf("memory pointer position = %d, want 0", pointer.Position)
	}

	position, err := store.NewPosition()
	if err != nil {
		t.Fatalf("new position: %v", err)
	}
	if position != 0 {
		t.Fatalf("memory new position = %d, want 0", position)
	}

	node, token, err := store.BeginAccess(pointer, position)
	if err != nil {
		t.Fatalf("begin access: %v", err)
	}
	if node.Value != "first" {
		t.Fatalf("unexpected value: %q", node.Value)
	}

	node.Value = "updated"
	if err := store.FinishAccess(token, node); err != nil {
		t.Fatalf("finish access: %v", err)
	}

	node, _, err = store.BeginAccess(pointer, position)
	if err != nil {
		t.Fatalf("begin updated access: %v", err)
	}
	if node.Value != "updated" {
		t.Fatalf("unexpected updated value: %q", node.Value)
	}
}

func TestStoreRejectsUnknownNode(t *testing.T) {
	store := New[string]()
	unknown := avl.Pointer{Address: 42}

	if _, _, err := store.BeginAccess(unknown, 0); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("begin error = %v, want %v", err, ErrNodeNotFound)
	}
	if err := store.FinishAccess(
		avl.AccessToken{Address: unknown.Address},
		avl.Node[string]{},
	); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("finish error = %v, want %v", err, ErrNodeNotFound)
	}
	if err := store.FreeMaybe(unknown, 1); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("free error = %v, want %v", err, ErrNodeNotFound)
	}
}

func TestStoreFreeMaybe(t *testing.T) {
	store := New[string]()
	pointer, err := store.Allocate(avl.Node[string]{Value: "candidate"})
	if err != nil {
		t.Fatalf("allocate node: %v", err)
	}

	if err := store.FreeMaybe(pointer, 0); err != nil {
		t.Fatalf("FreeMaybe(condition=0): %v", err)
	}
	if _, _, err := store.BeginAccess(pointer, 0); err != nil {
		t.Fatalf("candidate was freed with condition=0: %v", err)
	}

	if err := store.FreeMaybe(pointer, 1); err != nil {
		t.Fatalf("FreeMaybe(condition=1): %v", err)
	}
	if _, _, err := store.BeginAccess(pointer, 0); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("load freed node error = %v, want %v", err, ErrNodeNotFound)
	}

	if err := store.FreeMaybe(avl.Pointer{Address: 42}, 0); err != nil {
		t.Fatalf("FreeMaybe(unknown, condition=0): %v", err)
	}
}
