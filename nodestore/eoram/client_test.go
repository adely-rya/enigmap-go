package eoram

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/storage"
)

func TestNewOramClient(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)

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
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)

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
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	tests := []struct {
		name string
		call func()
	}{
		{
			name: "FinishAccess",
			call: func() {
				_ = client.FinishAccess(
					avl.AccessToken{Address: 1},
					avl.Node[selectableString]{Value: "value"},
				)
			},
		},
		{
			name: "Allocate",
			call: func() {
				_, _ = client.Allocate(
					avl.Node[selectableString]{Value: "value"},
				)
			},
		},
		{
			name: "FreeMaybe",
			call: func() {
				_ = client.FreeMaybe(avl.Pointer{Address: 1}, 1)
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

func TestClientBeginAccessAddsPathToStash(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	dummyBucket := Bucket[NodeBlock[selectableString]]{{}}
	for index := uint64(0); index < store.Size(); index++ {
		if err := store.Write(dummyBucket, index); err != nil {
			t.Fatalf("write dummy bucket %d: %v", index, err)
		}
	}

	target := Block[NodeBlock[selectableString]]{
		Address:  7,
		Position: 0,
		Data: NodeBlock[selectableString]{
			Node: avl.Node[selectableString]{Value: "target"},
		},
		Valid: 1,
	}
	if err := store.Write(
		Bucket[NodeBlock[selectableString]]{target},
		3,
	); err != nil {
		t.Fatalf("write target bucket: %v", err)
	}

	client, err := NewOramClient[selectableString](store, 1)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	_, _, err = client.BeginAccess(
		avl.Pointer{Address: 7, Position: 0},
		1,
	)
	if !errors.Is(err, ErrBeginAccessIncomplete) {
		t.Fatalf(
			"BeginAccess error = %v, want %v",
			err,
			ErrBeginAccessIncomplete,
		)
	}

	found := false
	for _, entry := range client.stash.blocks {
		if entry.Block.Valid == 1 && entry.Block.Address == 7 {
			found = true
			if entry.Block.Data.Node.Value != "target" {
				t.Fatalf(
					"stashed value = %q, want target",
					entry.Block.Data.Node.Value,
				)
			}
			if entry.Sticky != 0 {
				t.Fatalf("path block Sticky = %d, want 0", entry.Sticky)
			}
		}
	}
	if !found {
		t.Fatal("target block was not added to stash")
	}
}

func TestClientNewPositionUsesServerLeafCount(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
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
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
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
