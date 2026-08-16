package eoram

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/adely-rya/oram/core"
)

var ErrInvalidTreeSize = errors.New(
	"eoram: storage size must describe a complete binary tree",
)

// OramServer is the concrete Path ORAM server used by the ENIGMAP NodeStore.
type OramServer[T any, S core.IO[T]] struct {
	Storage   S
	leafCount uint64
}

const maxORAMPathLength = 64

// pathBatchIndices keeps the public root-to-leaf bucket indices inline. A
// uint64 ORAM can have at most 64 buckets on a path, so the backing array does
// not need a per-access heap allocation.
//
// The value is passed to batchStorage by value rather than passing a slice
// through an interface. Passing a slice backed by a local array through an
// interface makes the backing array escape to the heap.
type pathBatchIndices struct {
	values [maxORAMPathLength]uint64
	length int
}

type batchStorage[T any] interface {
	ReadBatch(indices []uint64) ([]T, error)
	WriteBatch(values []T, indices []uint64) error
}

// pathBatchStorage is an internal extension used by LargeBucketStorage. The
// public ReadBatch/WriteBatch shape above stays intact so external storage
// implementations keep their existing batching behavior.
type pathBatchStorage[T any] interface {
	readPathBatch(indices pathBatchIndices) ([]T, error)
	readPathBatchInto(indices pathBatchIndices, dst []T) error
	writePathBatch(values []T, indices pathBatchIndices) error
}

func heapPathIndices(bucketIndex uint64, pathLength int) []uint64 {
	indices := make([]uint64, pathLength)
	for index := pathLength; index > 0; index-- {
		indices[index-1] = bucketIndex
		if index > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}
	return indices
}

// NewOramServer creates an ORAM server backed by storage.
func NewOramServer[T any, S core.IO[T]](storage S) (OramServer[T, S], error) {
	bucketCount := storage.Size()
	leafCount := bucketCount/2 + 1

	if leafCount == 0 || leafCount&(leafCount-1) != 0 || 2*leafCount-1 != bucketCount {
		return OramServer[T, S]{}, ErrInvalidTreeSize
	}

	return OramServer[T, S]{
		Storage:   storage,
		leafCount: leafCount,
	}, nil
}

// LeafCount returns the fixed number of leaves in the ORAM tree.
func (server *OramServer[T, S]) LeafCount() uint64 {
	return server.leafCount
}

func (server *OramServer[T, S]) pathInfo(leaf Leaf) (uint64, int, error) {
	leafCount := server.LeafCount()
	if leafCount == 0 {
		return 0, 0, fmt.Errorf("tree has no leaves")
	}

	if uint64(leaf) >= leafCount {
		return 0, 0, fmt.Errorf(
			"leaf index out of range: leaf=%d, range=[0, %d]",
			leaf,
			leafCount-1,
		)
	}

	if leafCount&(leafCount-1) != 0 {
		return 0, 0, fmt.Errorf("number of leaves must be a power of two: N=%d", leafCount)
	}

	return (leafCount - 1) + uint64(leaf), bits.Len64(leafCount), nil
}

func (server *OramServer[T, S]) ReadPath(leaf Leaf) ([]T, error) {
	bucketIndex, pathLength, err := server.pathInfo(leaf)
	if err != nil {
		return nil, err
	}

	if pathLength > maxORAMPathLength {
		return nil, fmt.Errorf(
			"path length exceeds uint64 ORAM bound: got=%d max=%d",
			pathLength,
			maxORAMPathLength,
		)
	}

	leafBucketIndex := bucketIndex
	var indices pathBatchIndices
	indices.length = pathLength
	for pathIndex := pathLength; pathIndex > 0; pathIndex-- {
		indices.values[pathIndex-1] = bucketIndex
		if pathIndex > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	if storage, ok := any(server.Storage).(pathBatchStorage[T]); ok {
		return storage.readPathBatch(indices)
	}
	if storage, ok := any(server.Storage).(batchStorage[T]); ok {
		return storage.ReadBatch(heapPathIndices(leafBucketIndex, pathLength))
	}

	path := make([]T, pathLength)
	for index, bucketIndex := range indices.values[:indices.length] {
		bucket, err := server.Storage.Read(bucketIndex)
		if err != nil {
			return nil, err
		}
		path[index] = bucket
	}

	return path, nil
}

// readBucketPathInto is the slice-safe Client fast path. A generic core.IO may
// return a Bucket whose backing array is owned by the storage (MemoryStorage is
// one example), so assigning that slice into reusable scratch would alias the
// backing store. Copying block elements preserves the core.IO contract while
// still avoiding per-access Bucket allocations.
func readBucketPathInto[V core.Selectable[V]](
	server *OramServer[Bucket[V], core.IO[Bucket[V]]],
	leaf Leaf,
	dst []Bucket[V],
) ([]Bucket[V], error) {
	bucketIndex, pathLength, err := server.pathInfo(leaf)
	if err != nil {
		return nil, err
	}
	if pathLength > maxORAMPathLength {
		return nil, fmt.Errorf(
			"path length exceeds uint64 ORAM bound: got=%d max=%d",
			pathLength,
			maxORAMPathLength,
		)
	}
	if len(dst) < pathLength {
		return nil, fmt.Errorf(
			"path destination too small: got=%d want=%d",
			len(dst),
			pathLength,
		)
	}

	leafBucketIndex := bucketIndex
	var indices pathBatchIndices
	indices.length = pathLength
	for pathIndex := pathLength; pathIndex > 0; pathIndex-- {
		indices.values[pathIndex-1] = bucketIndex
		if pathIndex > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	path := dst[:pathLength]
	if storage, ok := any(server.Storage).(pathBatchStorage[Bucket[V]]); ok {
		if err := storage.readPathBatchInto(indices, path); err != nil {
			return nil, err
		}
		return path, nil
	}
	if storage, ok := any(server.Storage).(batchStorage[Bucket[V]]); ok {
		buckets, err := storage.ReadBatch(heapPathIndices(leafBucketIndex, pathLength))
		if err != nil {
			return nil, err
		}
		if len(buckets) != pathLength {
			return nil, fmt.Errorf(
				"batch path length mismatch: got=%d want=%d",
				len(buckets),
				pathLength,
			)
		}
		for index := range buckets {
			path[index] = copyBucketInto(path[index], buckets[index])
		}
		return path, nil
	}

	for index, logicalIndex := range indices.values[:indices.length] {
		bucket, err := server.Storage.Read(logicalIndex)
		if err != nil {
			return nil, err
		}
		path[index] = copyBucketInto(path[index], bucket)
	}
	return path, nil
}

// writeBucketPathFromScratch writes a reusable Client path without exposing
// its backing arrays to an arbitrary core.IO implementation. Optimized batch
// storage copies into its own cache-owned buffers. The generic fallback clones
// each Bucket because core.IO.Write is allowed to retain the slice value.
func writeBucketPathFromScratch[V core.Selectable[V]](
	server *OramServer[Bucket[V], core.IO[Bucket[V]]],
	path []Bucket[V],
	leaf Leaf,
) error {
	bucketIndex, pathLength, err := server.pathInfo(leaf)
	if err != nil {
		return err
	}
	if len(path) != pathLength {
		return fmt.Errorf(
			"path length mismatch: got=%d want=%d",
			len(path),
			pathLength,
		)
	}
	if pathLength > maxORAMPathLength {
		return fmt.Errorf(
			"path length exceeds uint64 ORAM bound: got=%d max=%d",
			pathLength,
			maxORAMPathLength,
		)
	}

	leafBucketIndex := bucketIndex
	var indices pathBatchIndices
	indices.length = pathLength
	for pathIndex := pathLength; pathIndex > 0; pathIndex-- {
		indices.values[pathIndex-1] = bucketIndex
		if pathIndex > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	if storage, ok := any(server.Storage).(pathBatchStorage[Bucket[V]]); ok {
		return storage.writePathBatch(path, indices)
	}
	if storage, ok := any(server.Storage).(batchStorage[Bucket[V]]); ok {
		ownedPath := cloneBucketPath(path)
		return storage.WriteBatch(
			ownedPath,
			heapPathIndices(leafBucketIndex, pathLength),
		)
	}
	for index, logicalIndex := range indices.values[:indices.length] {
		if err := server.Storage.Write(cloneBucket(path[index]), logicalIndex); err != nil {
			return err
		}
	}
	return nil
}

// WritePath writes a root-to-leaf path back to the ORAM tree.
func (server *OramServer[T, S]) WritePath(
	path []T,
	leaf Leaf,
) error {
	bucketIndex, pathLength, err := server.pathInfo(leaf)
	if err != nil {
		return err
	}

	if len(path) != pathLength {
		return fmt.Errorf(
			"path length mismatch: got=%d want=%d",
			len(path),
			pathLength,
		)
	}

	if pathLength > maxORAMPathLength {
		return fmt.Errorf(
			"path length exceeds uint64 ORAM bound: got=%d max=%d",
			pathLength,
			maxORAMPathLength,
		)
	}

	leafBucketIndex := bucketIndex
	var indices pathBatchIndices
	indices.length = pathLength
	for index := pathLength; index > 0; index-- {
		indices.values[index-1] = bucketIndex
		if index > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	if storage, ok := any(server.Storage).(pathBatchStorage[T]); ok {
		return storage.writePathBatch(path, indices)
	}
	if storage, ok := any(server.Storage).(batchStorage[T]); ok {
		return storage.WriteBatch(path, heapPathIndices(leafBucketIndex, pathLength))
	}

	for index, bucketIndex := range indices.values[:indices.length] {
		if err := server.Storage.Write(path[index], bucketIndex); err != nil {
			return err
		}
	}

	return nil
}
