package eoram

import (
	"testing"

	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/oram/pathoram"
	"github.com/adely-rya/oram/storage"
)

func TestNewOramClient(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[selectableString]](7)
	server := OramServer[
		Bucket[selectableString],
		core.IO[Bucket[selectableString]],
	]{
		Storage: store,
	}

	client := NewOramClient(server, 3)

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
	server := OramServer[
		Bucket[selectableString],
		core.IO[Bucket[selectableString]],
	]{
		Storage: store,
	}

	client := NewOramClient(server, 0)

	if client.bucketSize != pathoram.BUCKET_SIZE {
		t.Fatalf(
			"bucket size = %d, want %d",
			client.bucketSize,
			pathoram.BUCKET_SIZE,
		)
	}
}
