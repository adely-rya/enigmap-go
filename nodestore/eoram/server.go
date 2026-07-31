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

type batchStorage[T any] interface {
	ReadBatch(indices []uint64) ([]T, error)
	WriteBatch(values []T, indices []uint64) error
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

	indices := make([]uint64, pathLength)
	for pathIndex := pathLength; pathIndex > 0; pathIndex-- {
		indices[pathIndex-1] = bucketIndex
		if pathIndex > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	if storage, ok := any(server.Storage).(batchStorage[T]); ok {
		return storage.ReadBatch(indices)
	}

	path := make([]T, pathLength)
	for index, bucketIndex := range indices {
		bucket, err := server.Storage.Read(bucketIndex)
		if err != nil {
			return nil, err
		}
		path[index] = bucket
	}

	return path, nil
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

	indices := make([]uint64, pathLength)
	for index := pathLength; index > 0; index-- {
		indices[index-1] = bucketIndex
		if index > 1 {
			bucketIndex = (bucketIndex - 1) / 2
		}
	}

	if storage, ok := any(server.Storage).(batchStorage[T]); ok {
		return storage.WriteBatch(path, indices)
	}

	for index, bucketIndex := range indices {
		if err := server.Storage.Write(path[index], bucketIndex); err != nil {
			return err
		}
	}

	return nil
}
