package eoram

import (
	"testing"

	"github.com/adely-rya/oram/storage"
)

func TestNewOramServer(t *testing.T) {
	store := storage.NewMemoryStorage[int](7)

	server := NewOramServer(store)

	if server.Storage != store {
		t.Fatal("server does not retain storage")
	}
}
