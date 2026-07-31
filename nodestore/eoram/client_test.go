package eoram

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/adely-rya/oram/storage"
)

func TestNewOramClient(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)

	client, err := NewOramClient[selectableString](store, 3)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

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

	client, err := NewOramClient[selectableString](store, 0)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	if client.bucketSize != defaultBucketSize {
		t.Fatalf(
			"bucket size = %d, want %d",
			client.bucketSize,
			defaultBucketSize,
		)
	}
}

func TestClientOperationStubsPanic(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	tests := []struct {
		name string
		call func()
	}{
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

func TestClientNewPositionUsesServerLeafCount(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.random = bytes.NewReader([]byte{
		0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	})

	position, err := client.NewPosition()
	if err != nil {
		t.Fatalf("NewPosition: %v", err)
	}
	if position != 3 {
		t.Fatalf("position = %d, want 3", position)
	}
}

func TestClientNewPositionPropagatesRandomError(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.random = bytes.NewReader(nil)

	_, err = client.NewPosition()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("NewPosition error = %v, want EOF", err)
	}
}
