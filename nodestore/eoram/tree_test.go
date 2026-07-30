package eoram

import (
	"errors"
	"testing"

	"github.com/adely-rya/oram/oram/pathoram"
	"github.com/adely-rya/oram/storage"
)

func TestNewTreeParametersRoundsLeavesToPowerOfTwo(t *testing.T) {
	parameters, err := newTreeParameters(100)
	if err != nil {
		t.Fatalf("newTreeParameters() error = %v", err)
	}

	if parameters.logicalBlockCount != 102 {
		t.Fatalf("logicalBlockCount = %d, want 102", parameters.logicalBlockCount)
	}
	if parameters.leafCount != 128 {
		t.Fatalf("leafCount = %d, want 128", parameters.leafCount)
	}
	if parameters.bucketCount != 255 {
		t.Fatalf("bucketCount = %d, want 255", parameters.bucketCount)
	}
	if parameters.height != 7 {
		t.Fatalf("height = %d, want 7", parameters.height)
	}
}

func TestInitializeTreeWritesEmptyBucketToEveryIndex(t *testing.T) {
	const bucketCount = 15
	store := storage.NewMemoryStorage[pathoram.Bucket[nodeBlock]](bucketCount)

	if err := initializeTree(store); err != nil {
		t.Fatalf("initializeTree() error = %v", err)
	}

	for index := uint64(0); index < bucketCount; index++ {
		bucket, err := store.Read(index)
		if err != nil {
			t.Fatalf("Read(%d) error = %v", index, err)
		}
		if len(bucket) != pathoram.BUCKET_SIZE {
			t.Fatalf(
				"Read(%d) bucket length = %d, want %d",
				index,
				len(bucket),
				pathoram.BUCKET_SIZE,
			)
		}
	}
}

type failingBucketIO struct {
	size        uint64
	failingAt   uint64
	expectedErr error
}

func (store *failingBucketIO) Size() uint64 {
	return store.size
}

func (store *failingBucketIO) Read(
	index uint64,
) (pathoram.Bucket[nodeBlock], error) {
	return nil, nil
}

func (store *failingBucketIO) Write(
	bucket pathoram.Bucket[nodeBlock],
	index uint64,
) error {
	if index == store.failingAt {
		return store.expectedErr
	}
	return nil
}

func TestInitializeTreeStopsAtWriteError(t *testing.T) {
	expectedErr := errors.New("write failed")
	store := &failingBucketIO{
		size:        4,
		failingAt:   2,
		expectedErr: expectedErr,
	}

	err := initializeTree(store)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("initializeTree() error = %v, want %v", err, expectedErr)
	}
}
