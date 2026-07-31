package eoram

import (
	"bytes"
	"errors"
	"fmt"
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
	if client.stash.capacity == 0 || len(client.stash.blocks) != 0 {
		t.Fatalf("dynamic stash = (len=%d, capacity=%d), want empty with positive capacity", len(client.stash.blocks), client.stash.capacity)
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

func TestClientUncacheAllUnpinsEveryStashEntry(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.stash.blocks = append(
		client.stash.blocks,
		Block[NodeBlock[selectableString]]{Valid: 1}.MakeStashedBlock(1),
		Block[NodeBlock[selectableString]]{Valid: 1}.MakeStashedBlock(1),
	)

	client.UncacheAll()

	for index, entry := range client.stash.blocks {
		if entry.Sticky != 0 {
			t.Fatalf("stash entry %d Sticky = %d, want 0", index, entry.Sticky)
		}
	}
}

func TestClientAllocateAddsPinnedBlockToStash(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.random = bytes.NewReader(make([]byte, 8))

	pointer, err := client.Allocate(avl.Node[selectableString]{
		Value: "allocated",
		Valid: 1,
	})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if pointer.Address != 1 {
		t.Fatalf("pointer address = %d, want 1", pointer.Address)
	}
	if pointer.Position != 0 {
		t.Fatalf("pointer position = %d, want 0", pointer.Position)
	}

	found := false
	for _, entry := range client.stash.blocks {
		if entry.Block.Valid == 0 || entry.Block.BlockAddress != 1 {
			continue
		}

		found = true
		if entry.Block.Data.Node.Value != "allocated" {
			t.Fatalf("allocated value = %q, want allocated", entry.Block.Data.Node.Value)
		}
		if entry.Sticky != 1 {
			t.Fatalf("allocated block Sticky = %d, want 1", entry.Sticky)
		}
	}
	if !found {
		t.Fatal("allocated block is missing from stash")
	}
	if client.nextFreeBlock != 2 {
		t.Fatalf("next free block = %d, want 2", client.nextFreeBlock)
	}
}

func TestClientFreeMaybe(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.stash.blocks = append(client.stash.blocks, Block[NodeBlock[selectableString]]{
		BlockAddress: 9,
		Leaf:         2,
		Data: NodeBlock[selectableString]{
			Node: avl.Node[selectableString]{Value: "candidate", Valid: 1},
		},
		Valid: 1,
	}.MakeStashedBlock(1))

	if err := client.FreeMaybe(avl.Pointer{Address: 9}, 0); err != nil {
		t.Fatalf("FreeMaybe(condition=0): %v", err)
	}
	if client.stash.blocks[0].Block.Valid != 1 || client.stash.blocks[0].Sticky != 1 {
		t.Fatalf("condition=0 changed block: %+v", client.stash.blocks[0])
	}

	if err := client.FreeMaybe(avl.Pointer{Address: 9}, 1); err != nil {
		t.Fatalf("FreeMaybe(condition=1): %v", err)
	}
	if client.stash.blocks[0].Block.Valid != 0 {
		t.Fatalf("freed block Valid = %d, want 0", client.stash.blocks[0].Block.Valid)
	}
	if client.stash.blocks[0].Sticky != 0 {
		t.Fatalf("freed block Sticky = %d, want 0", client.stash.blocks[0].Sticky)
	}
	if client.stash.blocks[0].Block.Data.Node.Value != "" {
		t.Fatalf(
			"freed block data = %q, want zero value",
			client.stash.blocks[0].Block.Data.Node.Value,
		)
	}
}

func TestClientFreeMaybeRejectsMissingBlock(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	err = client.FreeMaybe(avl.Pointer{Address: 9}, 1)
	if !errors.Is(err, ErrBlockNotFound) {
		t.Fatalf("FreeMaybe error = %v, want %v", err, ErrBlockNotFound)
	}
}

func TestClientFinishAccessRejectsWhenNoAccessIsOpen(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	err = client.FinishAccess(
		avl.AccessToken{Address: 1},
		avl.Node[selectableString]{},
	)
	if !errors.Is(err, ErrNoAccessOpen) {
		t.Fatalf("FinishAccess error = %v, want %v", err, ErrNoAccessOpen)
	}
}

func TestClientFinishAccessRejectsMismatchedToken(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 5)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	client.access = accessState{
		phase:        accessOpen,
		blockAddress: 1,
	}

	err = client.FinishAccess(
		avl.AccessToken{Address: 2},
		avl.Node[selectableString]{},
	)
	if !errors.Is(err, ErrAccessTokenMismatch) {
		t.Fatalf(
			"FinishAccess error = %v, want %v",
			err,
			ErrAccessTokenMismatch,
		)
	}
}

func TestClientBeginAccessAddsPathToStash(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)
	client, err := NewOramClient[selectableString](store, 1)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	target := Block[NodeBlock[selectableString]]{
		BlockAddress: 7,
		Leaf:         0,
		Data: NodeBlock[selectableString]{
			Node: avl.Node[selectableString]{Value: "target", Valid: 1},
		},
		Valid: 1,
	}
	if err := store.Write(
		Bucket[NodeBlock[selectableString]]{target},
		3,
	); err != nil {
		t.Fatalf("write target bucket: %v", err)
	}

	node, token, err := client.BeginAccess(
		avl.Pointer{Address: 7, Position: 0},
		1,
	)
	if err != nil {
		t.Fatalf("BeginAccess: %v", err)
	}
	if node.Value != "target" {
		t.Fatalf("BeginAccess node value = %q, want target", node.Value)
	}
	if token.Address != 7 {
		t.Fatalf("BeginAccess token address = %d, want 7", token.Address)
	}

	found := false
	for _, entry := range client.stash.blocks {
		if entry.Block.Valid == 1 && entry.Block.BlockAddress == 7 {
			found = true
			if entry.Block.Data.Node.Value != "target" {
				t.Fatalf(
					"stashed value = %q, want target",
					entry.Block.Data.Node.Value,
				)
			}
			if entry.Block.Leaf != 1 {
				t.Fatalf("path block Leaf = %d, want 1", entry.Block.Leaf)
			}
			if entry.Sticky != 1 {
				t.Fatalf("path block Sticky = %d, want 1", entry.Sticky)
			}
		}
	}
	if !found {
		t.Fatal("target block was not added to stash")
	}
	if client.access.phase != accessOpen {
		t.Fatalf("access phase = %d, want open", client.access.phase)
	}
	if client.access.blockAddress != 7 {
		t.Fatalf("access block address = %d, want 7", client.access.blockAddress)
	}
	if client.access.oldLeaf != 0 {
		t.Fatalf("access old leaf = %d, want 0", client.access.oldLeaf)
	}
	if client.access.newLeaf != 1 {
		t.Fatalf("access new leaf = %d, want 1", client.access.newLeaf)
	}
	if client.access.pathLength != 3 {
		t.Fatalf("access path length = %d, want 3", client.access.pathLength)
	}

	_, _, err = client.BeginAccess(
		avl.Pointer{Address: 7, Position: 0},
		2,
	)
	if !errors.Is(err, ErrAccessAlreadyOpen) {
		t.Fatalf("second BeginAccess error = %v, want %v", err, ErrAccessAlreadyOpen)
	}

	node.Value = "updated"
	if err := client.FinishAccess(token, node); err != nil {
		t.Fatalf("FinishAccess: %v", err)
	}
	if client.access.phase != accessReady {
		t.Fatalf("access phase after FinishAccess = %d, want ready", client.access.phase)
	}

	value, stashFound := client.stash.Get(7)
	if stashFound != 1 {
		t.Fatal("updated block is missing from stash")
	}
	if value.Node.Value != "updated" {
		t.Fatalf("updated node value = %q, want updated", value.Node.Value)
	}
	for _, entry := range client.stash.blocks {
		if entry.Block.Valid == 1 && entry.Block.BlockAddress == 7 && entry.Sticky != 0 {
			t.Fatalf("updated block Sticky = %d, want 0", entry.Sticky)
		}
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

func TestClientBacksAVLTree(t *testing.T) {
	storage := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](31)
	client, err := NewOramClient[selectableString](storage, BucketSize)
	if err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}
	tree, err := avl.NewTree[selectableString](client, 8)
	if err != nil {
		t.Fatalf("NewTree: %v", err)
	}

	for index := 0; index < 10; index++ {
		key := make([]byte, avl.KeySize)
		copy(key, fmt.Sprintf("key-%02d", index))
		if err := tree.Put(key, selectableString(fmt.Sprintf("value-%02d", index))); err != nil {
			t.Fatalf("Put(%d): %v", index, err)
		}
	}
}
