package eoram

import (
	"errors"

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
func NewOramServer[T any, S core.IO[T]](
	storage S,
) (OramServer[T, S], error) {
	bucketCount := storage.Size()
	leafCount := bucketCount/2 + 1

	if leafCount == 0 ||
		leafCount&(leafCount-1) != 0 ||
		2*leafCount-1 != bucketCount {
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
