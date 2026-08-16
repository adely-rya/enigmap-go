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

type publicBatchIntStorage struct {
	storage      *storage.MemoryStorage[int]
	readBatches  int
	writeBatches int
}

func (storage *publicBatchIntStorage) Size() uint64 { return storage.storage.Size() }

func (storage *publicBatchIntStorage) Read(index uint64) (int, error) {
	return storage.storage.Read(index)
}

func (storage *publicBatchIntStorage) Write(value int, index uint64) error {
	return storage.storage.Write(value, index)
}

func (storage *publicBatchIntStorage) ReadBatch(indices []uint64) ([]int, error) {
	storage.readBatches++
	values := make([]int, len(indices))
	for index, storageIndex := range indices {
		value, err := storage.storage.Read(storageIndex)
		if err != nil {
			return nil, err
		}
		values[index] = value
	}
	return values, nil
}

func (storage *publicBatchIntStorage) WriteBatch(values []int, indices []uint64) error {
	storage.writeBatches++
	for index, storageIndex := range indices {
		if err := storage.storage.Write(values[index], storageIndex); err != nil {
			return err
		}
	}
	return nil
}

func TestOramServerPreservesPublicBatchStorageFastPath(t *testing.T) {
	backing := storage.NewMemoryStorage[int](7)
	for index := uint64(0); index < backing.Size(); index++ {
		if err := backing.Write(int(index), index); err != nil {
			t.Fatalf("write index %d: %v", index, err)
		}
	}
	batched := &publicBatchIntStorage{storage: backing}
	server, err := NewOramServer[int](batched)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}

	path, err := server.ReadPath(2)
	if err != nil {
		t.Fatalf("ReadPath: %v", err)
	}
	if batched.readBatches != 1 {
		t.Fatalf("ReadPath batch calls = %d, want 1", batched.readBatches)
	}
	want := []int{0, 2, 5}
	for index := range want {
		if path[index] != want[index] {
			t.Fatalf("path[%d] = %d, want %d", index, path[index], want[index])
		}
	}

	if err := server.WritePath([]int{10, 20, 30}, 2); err != nil {
		t.Fatalf("WritePath: %v", err)
	}
	if batched.writeBatches != 1 {
		t.Fatalf("WritePath batch calls = %d, want 1", batched.writeBatches)
	}
}
