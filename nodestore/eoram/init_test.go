package eoram

import (
	"testing"

	"github.com/adely-rya/oram/storage"
)

func TestNewOramClientInitializesEmptyBuckets(t *testing.T) {
	store := storage.NewMemoryStorage[Bucket[NodeBlock[selectableString]]](7)

	if _, err := NewOramClient[selectableString](store, 3); err != nil {
		t.Fatalf("NewOramClient: %v", err)
	}

	for bucketIndex := uint64(0); bucketIndex < store.Size(); bucketIndex++ {
		bucket, err := store.Read(bucketIndex)
		if err != nil {
			t.Fatalf("read bucket %d: %v", bucketIndex, err)
		}
		if len(bucket) != 3 {
			t.Fatalf("bucket %d size = %d, want 3", bucketIndex, len(bucket))
		}
		for slot := range bucket {
			if bucket[slot].Valid != 0 {
				t.Fatalf("bucket %d slot %d is not dummy", bucketIndex, slot)
			}
			if bucket[slot].Data.Node.Valid != 0 {
				t.Fatalf(
					"bucket %d slot %d contains valid node data",
					bucketIndex,
					slot,
				)
			}
		}
	}
}
