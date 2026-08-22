package main

import (
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestORAMMemoryEncryptedPutGet(t *testing.T) {
	store, err := newORAMMemoryEncrypted(
		3,
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("newORAMMemoryEncrypted: %v", err)
	}

	key := make([]byte, avl.KeySize)
	copy(key, "key")
	want := exampleValue{0x7b}
	if err := store.Put(key, want); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, found, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || got != want {
		t.Fatalf("Get = (%v, %t), want (%v, true)", got, found, want)
	}
}
