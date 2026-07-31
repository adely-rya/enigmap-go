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
