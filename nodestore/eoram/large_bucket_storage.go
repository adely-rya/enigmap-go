package eoram

import (
	"container/list"
	"fmt"

	"github.com/adely-rya/oram/core"
)

// LargeBucketStorage exposes logical Bucket I/O while storing packed
// LargeBuckets in the underlying physical storage.
type LargeBucketStorage[T core.Selectable[T]] struct {
	storage core.IO[LargeBucket[T]]
	layout  LargeBucketLayout

	cacheCapacity int
	cacheEntries  map[uint64]*list.Element
	recency       list.List
}

type largeBucketCacheEntry[T core.Selectable[T]] struct {
	physicalIndex uint64
	value         LargeBucket[T]
	dirty         bool
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
		cacheEntries:  make(map[uint64]*list.Element, cacheCapacity),
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
	return storage.WriteBatch([]Bucket[T]{bucket}, []uint64{index})
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
	for index, logicalIndex := range indices {
		location, err := storage.layout.Locate(logicalIndex)
		if err != nil {
			return nil, err
		}
		largeBucket, err := storage.load(location.PhysicalIndex)
		if err != nil {
			return nil, err
		}
		result[index] = cloneBucket(largeBucket[location.InnerIndex])
	}
	return result, nil
}

func (storage *LargeBucketStorage[T]) WriteBatch(buckets []Bucket[T], indices []uint64) error {
	if len(buckets) != len(indices) {
		return fmt.Errorf("eoram: batch length mismatch: buckets=%d indices=%d", len(buckets), len(indices))
	}

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
		value := cloneLargeBucket(entry.value)
		for _, update := range updates[physicalIndex] {
			value[update.innerIndex] = cloneBucket(update.bucket)
		}
		if !cached {
			if err := storage.storage.Write(value, physicalIndex); err != nil {
				return err
			}
			continue
		}
		entry.value = value
		entry.dirty = true
	}
	return nil
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
	if element, ok := storage.cacheEntries[physicalIndex]; ok {
		storage.recency.MoveToFront(element)
		entry := element.Value.(*largeBucketCacheEntry[T])
		return entry, true, nil
	}

	largeBucket, err := storage.storage.Read(physicalIndex)
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
		return &largeBucketCacheEntry[T]{
			physicalIndex: physicalIndex,
			value:         largeBucket,
		}, false, nil
	}

	if storage.recency.Len() == storage.cacheCapacity {
		if err := storage.evictLRU(); err != nil {
			return nil, false, err
		}
	}
	entry := &largeBucketCacheEntry[T]{
		physicalIndex: physicalIndex,
		value:         cloneLargeBucket(largeBucket),
	}
	element := storage.recency.PushFront(entry)
	storage.cacheEntries[physicalIndex] = element
	return entry, true, nil
}

func (storage *LargeBucketStorage[T]) evictLRU() error {
	victim := storage.recency.Back()
	if victim == nil {
		return nil
	}
	entry := victim.Value.(*largeBucketCacheEntry[T])
	if entry.dirty {
		if err := storage.storage.Write(entry.value, entry.physicalIndex); err != nil {
			return err
		}
	}
	delete(storage.cacheEntries, entry.physicalIndex)
	storage.recency.Remove(victim)
	return nil
}

func (storage *LargeBucketStorage[T]) clearCache() {
	clear(storage.cacheEntries)
	storage.recency.Init()
}

// Flush writes every dirty Large Bucket to the encrypted backing storage.
func (storage *LargeBucketStorage[T]) Flush() error {
	for element := storage.recency.Front(); element != nil; element = element.Next() {
		entry := element.Value.(*largeBucketCacheEntry[T])
		if !entry.dirty {
			continue
		}
		if err := storage.storage.Write(entry.value, entry.physicalIndex); err != nil {
			return err
		}
		entry.dirty = false
	}
	return nil
}

// FlushAndClear persists dirty entries and removes all cached plaintext.
func (storage *LargeBucketStorage[T]) FlushAndClear() error {
	if err := storage.Flush(); err != nil {
		return err
	}
	storage.clearCache()
	return nil
}
