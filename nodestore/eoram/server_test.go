package eoram

import (
	"errors"
	"testing"

	"github.com/adely-rya/oram/storage"
)

func TestNewOramServer(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)

	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	if server.Storage != store {
		t.Fatal("server does not retain storage")
	}
	if server.LeafCount() != 4 {
		t.Fatalf("leaf count = %d, want 4", server.LeafCount())
	}
}

func TestNewOramServerRejectsInvalidTreeSize(t *testing.T) {
	for _, size := range []uint64{0, 2, 6, 8} {
		store := storage.NewMemoryStorage[int](size)

		_, err := NewOramServer(store)

		if !errors.Is(err, ErrInvalidTreeSize) {
			t.Fatalf(
				"NewOramServer(size=%d) error = %v, want %v",
				size,
				err,
				ErrInvalidTreeSize,
			)
		}
	}
}

func TestOramServerReadPathReturnsRootToLeaf(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)
	for index := uint64(0); index < store.Size(); index++ {
		if err := store.Write(int(index), index); err != nil {
			t.Fatalf("write index %d: %v", index, err)
		}
	}

	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	path, err := server.ReadPath(2)
	if err != nil {
		t.Fatalf("ReadPath: %v", err)
	}
	want := []int{0, 2, 5}
	if len(path) != len(want) {
		t.Fatalf("path length = %d, want %d", len(path), len(want))
	}
	for index := range want {
		if path[index] != want[index] {
			t.Fatalf("path[%d] = %d, want %d", index, path[index], want[index])
		}
	}
}

func TestOramServerReadPathRejectsInvalidPosition(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)
	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	if _, err := server.ReadPath(4); err == nil {
		t.Fatal("ReadPath accepted out-of-range position")
	}
}

func TestOramServerWritePathWritesRootToLeaf(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)
	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	if err := server.WritePath([]int{10, 20, 30}, 2); err != nil {
		t.Fatalf("WritePath: %v", err)
	}

	want := map[uint64]int{
		0: 10,
		2: 20,
		5: 30,
	}
	for index, wantValue := range want {
		got, err := store.Read(index)
		if err != nil {
			t.Fatalf("read index %d: %v", index, err)
		}
		if got != wantValue {
			t.Fatalf("storage[%d] = %d, want %d", index, got, wantValue)
		}
	}
}

func TestOramServerWritePathRejectsLengthMismatch(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)
	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	if err := server.WritePath([]int{10, 20}, 2); err == nil {
		t.Fatal("WritePath accepted a path with the wrong length")
	}
}

func TestOramServerWritePathRejectsInvalidPosition(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)
	server, err := NewOramServer(store)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	if err := server.WritePath([]int{10, 20, 30}, 4); err == nil {
		t.Fatal("WritePath accepted out-of-range position")
	}
}
