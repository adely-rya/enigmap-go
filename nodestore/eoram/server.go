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

func (oram *OramServer[T, S]) pathInfo(leaf Position) (uint64, int, error) {
	leafCount := oram.LeafCount()
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

func (server *OramServer[T, S]) ReadPath(leafIndex Position) ([]T, error) {
	nodeIndex, pathLen, err := server.pathInfo(leafIndex)
	if err != nil {
		return nil, err
	}

	leaves := make([]T, pathLen)
	for i := pathLen; i > 0; i-- {
		leaf, err := server.Storage.Read(nodeIndex)
		if err != nil {
			return nil, err
		}

		leaves[i-1] = leaf
		if i > 1 {
			nodeIndex = (nodeIndex - 1) / 2
		}
	}

	return leaves, nil
}
