package eoram

import (
	"testing"

	"github.com/adely-rya/oram/storage"
)

type countingLargeBucketStorage struct {
	storage *storage.MemoryStorage[LargeBucket[serializablePayload]]
	reads   int
	writes  int
}

func (storage *countingLargeBucketStorage) Size() uint64 { return storage.storage.Size() }

func (storage *countingLargeBucketStorage) Read(index uint64) (LargeBucket[serializablePayload], error) {
	storage.reads++
	return storage.storage.Read(index)
}

func (storage *countingLargeBucketStorage) Write(
	value LargeBucket[serializablePayload],
	index uint64,
) error {
	storage.writes++
	return storage.storage.Write(value, index)
}

func TestLargeBucketStorageBatchesPathIO(t *testing.T) {
	layout, err := NewLargeBucketLayout(7)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	physical := &countingLargeBucketStorage{
		storage: storage.NewMemoryStorage[LargeBucket[serializablePayload]](
			layout.PhysicalBucketCount(),
		),
	}
	logical, err := NewLargeBucketStorage[serializablePayload](physical, layout, 2)
	if err != nil {
		t.Fatalf("NewLargeBucketStorage: %v", err)
	}

	empty := make(Bucket[serializablePayload], BucketSize)
	if err := logical.Initialize(empty); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	physical.reads = 0
	physical.writes = 0

	server, err := NewOramServer[Bucket[serializablePayload]](logical)
	if err != nil {
		t.Fatalf("NewOramServer: %v", err)
	}
	path, err := server.ReadPath(2)
	if err != nil {
		t.Fatalf("ReadPath: %v", err)
	}
	wantPathIO := (len(path) + LevelsPerLargeBucket - 1) / LevelsPerLargeBucket
	if physical.reads != wantPathIO {
		t.Fatalf("ReadPath physical reads = %d, want %d", physical.reads, wantPathIO)
	}

	physical.reads = 0
	physical.writes = 0
	path[0][0].Valid = 1
	path[1][0].Valid = 1
	path[2][0].Valid = 1
	if err := server.WritePath(path, 2); err != nil {
		t.Fatalf("WritePath: %v", err)
	}
	if physical.reads != 0 || physical.writes != 0 {
		t.Fatalf(
			"WritePath physical I/O = (%d reads, %d writes), want (0, 0)",
			physical.reads,
			physical.writes,
		)
	}
	if err := logical.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if physical.writes != wantPathIO {
		t.Fatalf("Flush physical writes = %d, want %d", physical.writes, wantPathIO)
	}
}

func TestLargeBucketStorageMovesHitsToMRU(t *testing.T) {
	logicalBucketCount := uint64((1 << (LevelsPerLargeBucket + 1)) - 1)
	layout, err := NewLargeBucketLayout(logicalBucketCount)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	physical := &countingLargeBucketStorage{
		storage: storage.NewMemoryStorage[LargeBucket[serializablePayload]](
			layout.PhysicalBucketCount(),
		),
	}
	logical, err := NewLargeBucketStorage[serializablePayload](physical, layout, 2)
	if err != nil {
		t.Fatalf("NewLargeBucketStorage: %v", err)
	}
	if err := logical.Initialize(make(Bucket[serializablePayload], BucketSize)); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	physical.reads = 0
	physical.writes = 0

	secondGroupStart := uint64((1 << LevelsPerLargeBucket) - 1)
	// These indexes belong to physical Large Buckets 0, 1, and 2.
	// Re-reading 0 makes it MRU, so loading 2 must evict 1.
	for _, logicalIndex := range []uint64{0, secondGroupStart, 0, secondGroupStart + 1, 0} {
		if _, err := logical.Read(logicalIndex); err != nil {
			t.Fatalf("Read(%d): %v", logicalIndex, err)
		}
	}
	if physical.reads != 3 {
		t.Fatalf("physical reads = %d, want 3", physical.reads)
	}

	if _, err := logical.Read(secondGroupStart); err != nil {
		t.Fatalf("Read(%d): %v", secondGroupStart, err)
	}
	if physical.reads != 4 {
		t.Fatalf("physical reads after LRU reload = %d, want 4", physical.reads)
	}
}

func TestLargeBucketStorageKeepsDirtyValueInCache(t *testing.T) {
	layout, err := NewLargeBucketLayout(7)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	physical := &countingLargeBucketStorage{
		storage: storage.NewMemoryStorage[LargeBucket[serializablePayload]](
			layout.PhysicalBucketCount(),
		),
	}
	logical, err := NewLargeBucketStorage[serializablePayload](physical, layout, 2)
	if err != nil {
		t.Fatalf("NewLargeBucketStorage: %v", err)
	}
	if err := logical.Initialize(make(Bucket[serializablePayload], BucketSize)); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	physical.reads = 0
	physical.writes = 0

	bucket, err := logical.Read(0)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	bucket[0].Valid = 1
	if err := logical.Write(bucket, 0); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := logical.Read(0); err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if physical.reads != 1 {
		t.Fatalf("physical reads = %d, want 1", physical.reads)
	}
	if physical.writes != 0 {
		t.Fatalf("physical writes before Flush = %d, want 0", physical.writes)
	}
	if err := logical.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if physical.writes != 1 {
		t.Fatalf("physical writes after Flush = %d, want 1", physical.writes)
	}
	if err := logical.Flush(); err != nil {
		t.Fatalf("second Flush: %v", err)
	}
	if physical.writes != 1 {
		t.Fatalf("clean Flush physical writes = %d, want 1", physical.writes)
	}
}

func TestLargeBucketStorageWritesDirtyLRUVictim(t *testing.T) {
	logicalBucketCount := uint64((1 << (LevelsPerLargeBucket + 1)) - 1)
	layout, err := NewLargeBucketLayout(logicalBucketCount)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	physical := &countingLargeBucketStorage{
		storage: storage.NewMemoryStorage[LargeBucket[serializablePayload]](
			layout.PhysicalBucketCount(),
		),
	}
	logical, err := NewLargeBucketStorage[serializablePayload](physical, layout, 1)
	if err != nil {
		t.Fatalf("NewLargeBucketStorage: %v", err)
	}
	if err := logical.Initialize(make(Bucket[serializablePayload], BucketSize)); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	physical.reads = 0
	physical.writes = 0

	bucket, err := logical.Read(0)
	if err != nil {
		t.Fatalf("Read(0): %v", err)
	}
	bucket[0].Valid = 1
	if err := logical.Write(bucket, 0); err != nil {
		t.Fatalf("Write(0): %v", err)
	}
	secondGroupStart := uint64((1 << LevelsPerLargeBucket) - 1)
	if _, err := logical.Read(secondGroupStart); err != nil {
		t.Fatalf("Read(%d): %v", secondGroupStart, err)
	}
	if physical.writes != 1 {
		t.Fatalf("dirty eviction physical writes = %d, want 1", physical.writes)
	}

	stored, err := physical.storage.Read(0)
	if err != nil {
		t.Fatalf("read stored Large Bucket: %v", err)
	}
	if stored[0][0].Valid != 1 {
		t.Fatal("dirty eviction did not persist updated bucket")
	}
}

func TestLargeBucketStorageReadBatchPreservesNonDefaultBucketSize(t *testing.T) {
	layout, err := NewLargeBucketLayout(7)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	physical := storage.NewMemoryStorage[LargeBucket[serializablePayload]](
		layout.PhysicalBucketCount(),
	)
	logical, err := NewLargeBucketStorage[serializablePayload](physical, layout, 2)
	if err != nil {
		t.Fatalf("NewLargeBucketStorage: %v", err)
	}
	const bucketSize = 3
	if err := logical.Initialize(make(Bucket[serializablePayload], bucketSize)); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	buckets, err := logical.ReadBatch([]uint64{0, 1, 3})
	if err != nil {
		t.Fatalf("ReadBatch: %v", err)
	}
	for index, bucket := range buckets {
		if len(bucket) != bucketSize {
			t.Fatalf("bucket %d length = %d, want %d", index, len(bucket), bucketSize)
		}
	}

	buckets[0][0].Valid = 1
	if buckets[1][0].Valid != 0 {
		t.Fatal("ReadBatch results unexpectedly alias each other")
	}
}
