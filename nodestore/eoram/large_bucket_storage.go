package eoram

import (
	"fmt"
	"time"

	"github.com/adely-rya/oram/core"
)

// LargeBucketStorage exposes logical Bucket I/O while storing packed
// LargeBuckets in the underlying physical storage.
type LargeBucketStorage[T core.Selectable[T]] struct {
	storage core.IO[LargeBucket[T]]
	layout  LargeBucketLayout

	cacheCapacity int
	cacheEntries  map[uint64]int
	cacheSlots    []largeBucketCacheEntry[T]
	cacheUsed     int
	mruSlot       int
	lruSlot       int
	uncached      largeBucketCacheEntry[T]
	stats         LargeBucketCacheStats
}

type LargeBucketCacheStats struct {
	Hits             uint64
	Misses           uint64
	BackingReads     uint64
	BackingWrites    uint64
	DirtyEvictions   uint64
	CleanEvictions   uint64
	FlushWrites      uint64
	BackingReadTime  time.Duration
	BackingWriteTime time.Duration
}

type largeBucketCacheEntry[T core.Selectable[T]] struct {
	physicalIndex uint64
	value         LargeBucket[T]
	dirty         bool
	active        bool
	newer         int
	older         int
}

func NewLargeBucketStorage[T core.Selectable[T]](
	storage core.IO[LargeBucket[T]],
	layout LargeBucketLayout,
	cacheCapacity int,
) (*LargeBucketStorage[T], error) {
	if storage.Size() != layout.PhysicalBucketCount() {
		return nil, fmt.Errorf(
			"eoram: physical storage size mismatch: got=%d want=%d",
			storage.Size(),
			layout.PhysicalBucketCount(),
		)
	}
	if cacheCapacity < 0 {
		return nil, fmt.Errorf("eoram: cache capacity must not be negative: %d", cacheCapacity)
	}
	return &LargeBucketStorage[T]{
		storage:       storage,
		layout:        layout,
		cacheCapacity: cacheCapacity,
		cacheEntries:  make(map[uint64]int, cacheCapacity),
		cacheSlots:    make([]largeBucketCacheEntry[T], cacheCapacity),
		mruSlot:       -1,
		lruSlot:       -1,
	}, nil
}

func (storage *LargeBucketStorage[T]) Size() uint64 {
	return storage.layout.LogicalBucketCount()
}

func (storage *LargeBucketStorage[T]) Read(index uint64) (Bucket[T], error) {
	location, err := storage.layout.Locate(index)
	if err != nil {
		return nil, err
	}
	largeBucket, err := storage.load(location.PhysicalIndex)
	if err != nil {
		return nil, err
	}
	if len(largeBucket) != BucketsPerLargeBucket {
		return nil, fmt.Errorf(
			"eoram: physical large bucket size mismatch: got=%d want=%d",
			len(largeBucket),
			BucketsPerLargeBucket,
		)
	}
	return cloneBucket(largeBucket[location.InnerIndex]), nil
}

func (storage *LargeBucketStorage[T]) Write(bucket Bucket[T], index uint64) error {
	var buckets [1]Bucket[T]
	var indices [1]uint64
	buckets[0] = bucket
	indices[0] = index
	return storage.WriteBatch(buckets[:], indices[:])
}

func (storage *LargeBucketStorage[T]) Initialize(bucket Bucket[T]) error {
	storage.clearCache()
	empty := NewLargeBucket(bucket)
	for index := uint64(0); index < storage.storage.Size(); index++ {
		if err := storage.storage.Write(cloneLargeBucket(empty), index); err != nil {
			return err
		}
	}
	return nil
}

func (storage *LargeBucketStorage[T]) ReadBatch(indices []uint64) ([]Bucket[T], error) {
	result := make([]Bucket[T], len(indices))
	var blocks []Block[T]
	bucketSize := -1
	for index, logicalIndex := range indices {
		location, err := storage.layout.Locate(logicalIndex)
		if err != nil {
			return nil, err
		}
		largeBucket, err := storage.load(location.PhysicalIndex)
		if err != nil {
			return nil, err
		}
		source := largeBucket[location.InnerIndex]
		if bucketSize < 0 {
			bucketSize = len(source)
			blocks = make([]Block[T], len(indices)*bucketSize)
		}
		if len(source) == bucketSize {
			start := index * bucketSize
			result[index] = blocks[start : start+bucketSize]
			copy(result[index], source)
		} else {
			// Preserve support for irregular generic callers. Normal ORAM paths
			// are uniform and therefore share one contiguous backing allocation.
			result[index] = cloneBucket(source)
		}
	}
	return result, nil
}

func (storage *LargeBucketStorage[T]) readPathBatch(
	indices pathBatchIndices,
) ([]Bucket[T], error) {
	return storage.ReadBatch(indices.values[:indices.length])
}

func (storage *LargeBucketStorage[T]) readPathBatchInto(
	indices pathBatchIndices,
	dst []Bucket[T],
) error {
	if len(dst) != indices.length {
		return fmt.Errorf(
			"eoram: path destination length mismatch: got=%d want=%d",
			len(dst),
			indices.length,
		)
	}
	for index, logicalIndex := range indices.values[:indices.length] {
		location, err := storage.layout.Locate(logicalIndex)
		if err != nil {
			return err
		}
		largeBucket, err := storage.load(location.PhysicalIndex)
		if err != nil {
			return err
		}
		if len(largeBucket) != BucketsPerLargeBucket {
			return fmt.Errorf(
				"eoram: physical large bucket size mismatch: got=%d want=%d",
				len(largeBucket),
				BucketsPerLargeBucket,
			)
		}
		dst[index] = copyBucketInto(dst[index], largeBucket[location.InnerIndex])
	}
	return nil
}

func (storage *LargeBucketStorage[T]) WriteBatch(buckets []Bucket[T], indices []uint64) error {
	if len(buckets) != len(indices) {
		return fmt.Errorf("eoram: batch length mismatch: buckets=%d indices=%d", len(buckets), len(indices))
	}
	if len(indices) <= maxORAMPathLength {
		return storage.writeSmallBatch(buckets, indices)
	}
	return storage.writeLargeBatch(buckets, indices)
}

// writeSmallBatch is the hot ORAM-path case. Path batches are bounded by the
// 64 levels representable by a uint64 leaf, so grouping metadata can live in
// fixed local arrays instead of a map plus append-grown slices.
func (storage *LargeBucketStorage[T]) writeSmallBatch(
	buckets []Bucket[T],
	indices []uint64,
) error {
	var locations [maxORAMPathLength]LargeBucketLocation
	var physicalOrder [maxORAMPathLength]uint64
	physicalCount := 0

	for index, logicalIndex := range indices {
		location, err := storage.layout.Locate(logicalIndex)
		if err != nil {
			return err
		}
		locations[index] = location

		seen := false
		for orderIndex := 0; orderIndex < physicalCount; orderIndex++ {
			if physicalOrder[orderIndex] == location.PhysicalIndex {
				seen = true
				break
			}
		}
		if !seen {
			physicalOrder[physicalCount] = location.PhysicalIndex
			physicalCount++
		}
	}

	for orderIndex := 0; orderIndex < physicalCount; orderIndex++ {
		physicalIndex := physicalOrder[orderIndex]
		entry, cached, err := storage.loadEntry(physicalIndex)
		if err != nil {
			return err
		}
		value := entry.value
		for updateIndex := range indices {
			location := locations[updateIndex]
			if location.PhysicalIndex != physicalIndex {
				continue
			}
			value[location.InnerIndex] = copyBucketInto(
				value[location.InnerIndex],
				buckets[updateIndex],
			)
		}
		if err := storage.finishBatchWrite(entry, cached, physicalIndex); err != nil {
			return err
		}
	}
	return nil
}

// writeLargeBatch preserves the general public WriteBatch behavior for callers
// that submit more entries than a single ORAM path can contain.
func (storage *LargeBucketStorage[T]) writeLargeBatch(
	buckets []Bucket[T],
	indices []uint64,
) error {

	type bucketUpdate struct {
		innerIndex int
		bucket     Bucket[T]
	}
	updates := make(map[uint64][]bucketUpdate)
	order := make([]uint64, 0)
	for index, logicalIndex := range indices {
		location, err := storage.layout.Locate(logicalIndex)
		if err != nil {
			return err
		}
		if _, ok := updates[location.PhysicalIndex]; !ok {
			order = append(order, location.PhysicalIndex)
		}
		updates[location.PhysicalIndex] = append(
			updates[location.PhysicalIndex],
			bucketUpdate{innerIndex: location.InnerIndex, bucket: buckets[index]},
		)
	}

	for _, physicalIndex := range order {
		entry, cached, err := storage.loadEntry(physicalIndex)
		if err != nil {
			return err
		}
		// loadEntry returns cache-owned storage (or a private miss value when
		// caching is disabled). Update it in place and copy only the logical
		// buckets supplied by the caller; cloning the whole Large Bucket here
		// multiplied memory traffic by every packed bucket.
		value := entry.value
		for _, update := range updates[physicalIndex] {
			value[update.innerIndex] = copyBucketInto(
				value[update.innerIndex],
				update.bucket,
			)
		}
		if err := storage.finishBatchWrite(entry, cached, physicalIndex); err != nil {
			return err
		}
	}
	return nil
}

func (storage *LargeBucketStorage[T]) finishBatchWrite(
	entry *largeBucketCacheEntry[T],
	cached bool,
	physicalIndex uint64,
) error {
	if !cached {
		started := time.Now()
		if err := storage.storage.Write(entry.value, physicalIndex); err != nil {
			storage.stats.BackingWriteTime += time.Since(started)
			storage.stats.BackingWrites++
			return err
		}
		storage.stats.BackingWriteTime += time.Since(started)
		storage.stats.BackingWrites++
		return nil
	}
	entry.dirty = true
	return nil
}

func (storage *LargeBucketStorage[T]) writePathBatch(
	buckets []Bucket[T],
	indices pathBatchIndices,
) error {
	return storage.WriteBatch(buckets, indices.values[:indices.length])
}

func (storage *LargeBucketStorage[T]) load(physicalIndex uint64) (LargeBucket[T], error) {
	entry, _, err := storage.loadEntry(physicalIndex)
	if err != nil {
		return nil, err
	}
	return entry.value, nil
}

func (storage *LargeBucketStorage[T]) loadEntry(
	physicalIndex uint64,
) (*largeBucketCacheEntry[T], bool, error) {
	if slot, ok := storage.cacheEntries[physicalIndex]; ok {
		storage.stats.Hits++
		storage.moveSlotToMRU(slot)
		return &storage.cacheSlots[slot], true, nil
	}

	storage.stats.Misses++
	started := time.Now()
	largeBucket, err := storage.storage.Read(physicalIndex)
	storage.stats.BackingReadTime += time.Since(started)
	storage.stats.BackingReads++
	if err != nil {
		return nil, false, err
	}
	if len(largeBucket) != BucketsPerLargeBucket {
		return nil, false, fmt.Errorf(
			"eoram: physical large bucket size mismatch: got=%d want=%d",
			len(largeBucket),
			BucketsPerLargeBucket,
		)
	}
	if storage.cacheCapacity == 0 {
		storage.uncached.physicalIndex = physicalIndex
		// A generic core.IO.Read may return storage-owned slice backing. Work on
		// an owned copy so an un-cached update cannot mutate the backing store
		// before Write is called.
		storage.uncached.value = cloneLargeBucket(largeBucket)
		storage.uncached.dirty = false
		storage.uncached.active = true
		storage.uncached.newer = -1
		storage.uncached.older = -1
		return &storage.uncached, false, nil
	}

	var slot int
	if storage.cacheUsed < storage.cacheCapacity {
		slot = storage.cacheUsed
		storage.cacheUsed++
		entry := &storage.cacheSlots[slot]
		entry.value = cloneLargeBucket(largeBucket)
		entry.active = true
		entry.newer = -1
		entry.older = -1
		storage.insertSlotMRU(slot)
	} else {
		slot = storage.lruSlot
		if slot < 0 {
			return nil, false, fmt.Errorf("eoram: LRU cache has no victim")
		}
		entry := &storage.cacheSlots[slot]
		if err := storage.persistEvictedEntry(entry); err != nil {
			return nil, false, err
		}
		delete(storage.cacheEntries, entry.physicalIndex)
		entry.value = copyLargeBucketInto(entry.value, largeBucket)
		entry.dirty = false
		storage.moveSlotToMRU(slot)
	}

	entry := &storage.cacheSlots[slot]
	entry.physicalIndex = physicalIndex
	entry.dirty = false
	entry.active = true
	storage.cacheEntries[physicalIndex] = slot
	return entry, true, nil
}

func (storage *LargeBucketStorage[T]) persistEvictedEntry(
	entry *largeBucketCacheEntry[T],
) error {
	if entry.dirty {
		started := time.Now()
		// The generic core.IO contract does not promise that Write copies slice
		// backed values. Keep the cache slot reusable by handing the backing
		// storage an owned clone.
		if err := storage.storage.Write(cloneLargeBucket(entry.value), entry.physicalIndex); err != nil {
			storage.stats.BackingWriteTime += time.Since(started)
			storage.stats.BackingWrites++
			return err
		}
		storage.stats.BackingWriteTime += time.Since(started)
		storage.stats.BackingWrites++
		storage.stats.DirtyEvictions++
	} else {
		storage.stats.CleanEvictions++
	}
	return nil
}

func (storage *LargeBucketStorage[T]) insertSlotMRU(slot int) {
	entry := &storage.cacheSlots[slot]
	entry.newer = -1
	entry.older = storage.mruSlot
	if storage.mruSlot >= 0 {
		storage.cacheSlots[storage.mruSlot].newer = slot
	} else {
		storage.lruSlot = slot
	}
	storage.mruSlot = slot
}

func (storage *LargeBucketStorage[T]) moveSlotToMRU(slot int) {
	if slot == storage.mruSlot {
		return
	}
	entry := &storage.cacheSlots[slot]
	newer := entry.newer
	older := entry.older
	if newer >= 0 {
		storage.cacheSlots[newer].older = older
	}
	if older >= 0 {
		storage.cacheSlots[older].newer = newer
	} else {
		storage.lruSlot = newer
	}

	entry.newer = -1
	entry.older = storage.mruSlot
	if storage.mruSlot >= 0 {
		storage.cacheSlots[storage.mruSlot].newer = slot
	} else {
		storage.lruSlot = slot
	}
	storage.mruSlot = slot
}

func (storage *LargeBucketStorage[T]) clearCache() {
	clear(storage.cacheEntries)
	var zero largeBucketCacheEntry[T]
	for index := 0; index < storage.cacheUsed; index++ {
		storage.cacheSlots[index] = zero
	}
	storage.cacheUsed = 0
	storage.mruSlot = -1
	storage.lruSlot = -1
	storage.uncached = zero
}

// Flush writes every dirty Large Bucket to the encrypted backing storage.
func (storage *LargeBucketStorage[T]) Flush() error {
	for slot := 0; slot < storage.cacheUsed; slot++ {
		entry := &storage.cacheSlots[slot]
		if !entry.active || !entry.dirty {
			continue
		}
		started := time.Now()
		if err := storage.storage.Write(cloneLargeBucket(entry.value), entry.physicalIndex); err != nil {
			storage.stats.BackingWriteTime += time.Since(started)
			storage.stats.BackingWrites++
			return err
		}
		storage.stats.BackingWriteTime += time.Since(started)
		storage.stats.BackingWrites++
		storage.stats.FlushWrites++
		entry.dirty = false
	}
	return nil
}

func (storage *LargeBucketStorage[T]) Stats() LargeBucketCacheStats {
	return storage.stats
}

// FlushAndClear persists dirty entries and removes all cached plaintext.
func (storage *LargeBucketStorage[T]) FlushAndClear() error {
	if err := storage.Flush(); err != nil {
		return err
	}
	storage.clearCache()
	return nil
}
