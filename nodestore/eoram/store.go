package eoram

import (
	"errors"

	"github.com/adely-rya/enigmap-go/codec"
)

var (
	ErrInvalidMaxNodes  = errors.New("eoram: maxNodes must be positive")
	ErrNilValueCodec    = errors.New("eoram: value codec is required")
	ErrInvalidValueSize = errors.New("eoram: value codec size must be positive")
)

const DefaultDataFile = "/dev/shm/enigmap/oram-large-buckets.db"

// Store is the ENIGMAP-specific AVL node store backed by Path ORAM.
//
// The constructor owns the stable configuration boundary. Path ORAM, Large
// Bucket cache, encryption, and file storage are added behind this type.
type Store[V any] struct {
	maxNodes uint64
	codec    codec.ValueCodec[V]
	dataFile string
}

// New creates the ENIGMAP-specific Path ORAM node store configuration.
func New[V any](
	maxNodes uint64,
	valueCodec codec.ValueCodec[V],
	dataFile string,
) (*Store[V], error) {
	if maxNodes == 0 {
		return nil, ErrInvalidMaxNodes
	}
	if valueCodec == nil {
		return nil, ErrNilValueCodec
	}
	if valueCodec.Size() == 0 {
		return nil, ErrInvalidValueSize
	}

	return &Store[V]{
		maxNodes: maxNodes,
		codec:    valueCodec,
		dataFile: dataFile,
	}, nil
}

// Default creates an eoram store using the default tmpfs data file.
func Default[V any](
	maxNodes uint64,
	valueCodec codec.ValueCodec[V],
) (*Store[V], error) {
	return New(maxNodes, valueCodec, DefaultDataFile)
}
