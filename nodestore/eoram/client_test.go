package eoram

import (
	"testing"

	"github.com/adely-rya/oram/oram/pathoram"
	"github.com/adely-rya/oram/storage"
)

func TestNewOramClient(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)

	client := NewOramClient[selectableString](store, 3)

	if client.server.Storage != store {
		t.Fatal("client does not retain server storage")
	}
	if client.bucketSize != 3 {
		t.Fatalf("bucket size = %d, want 3", client.bucketSize)
	}
	if len(client.stash.blocks) == 0 {
		t.Fatal("stash is not initialized")
	}
	if client.access.phase != accessReady {
		t.Fatalf("access phase = %d, want ready", client.access.phase)
	}
}

func TestNewOramClientUsesDefaultBucketSize(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)

	client := NewOramClient[selectableString](store, 0)

	if client.bucketSize != pathoram.BUCKET_SIZE {
		t.Fatalf(
			"bucket size = %d, want %d",
			client.bucketSize,
			pathoram.BUCKET_SIZE,
		)
	}
}

func TestClientOperationStubsPanic(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)
	client := NewOramClient[selectableString](store, 5)

	tests := []struct {
		name string
		call func()
	}{
		{
			name: "NewPosition",
			call: func() {
				_, _ = client.NewPosition()
			},
		},
		{
			name: "BeginAccess",
			call: func() {
				_, _ = client.BeginAccess(1, 2, 3, 1)
			},
		},
		{
			name: "FinishAccess",
			call: func() {
				_ = client.FinishAccess(1, "value", 1)
			},
		},
		{
			name: "Allocate",
			call: func() {
				_, _ = client.Allocate("value", 3)
			},
		},
		{
			name: "FreeMaybe",
			call: func() {
				_ = client.FreeMaybe(1, 1)
			},
		},
		{
			name: "UncacheAll",
			call: func() {
				client.UncacheAll()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("operation did not panic")
				}
			}()

			test.call()
		})
	}
}
