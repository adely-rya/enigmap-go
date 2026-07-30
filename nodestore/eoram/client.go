package eoram

import (
	"errors"

	"github.com/adely-rya/enigmap-go/codec"
)

var (
	ErrInvalidCapacity  = errors.New("eoram: capacity must be positive")
	ErrNilValueCodec    = errors.New("eoram: value codec is required")
	ErrInvalidValueSize = errors.New("eoram: value codec size must be positive")
)

const DefaultDataFile = "/dev/shm/enigmap/oram-large-buckets.db"

// ORAMClient is the ENIGMAP-specific AVL node store backed by Path ORAM.
//
// The constructor owns the stable configuration boundary. Path ORAM, Large
// Bucket cache, encryption, and file storage are added behind this type.
type ORAMClient[V any] struct {
	capacity uint64
	codec    codec.ValueCodec[V]
	dataFile string
}

// New creates the ENIGMAP-specific Path ORAM node store configuration.
func New[V any](
	capacity uint64,
	valueCodec codec.ValueCodec[V],
	dataFile string,
) (*ORAMClient[V], error) {
	if capacity == 0 {
		return nil, ErrInvalidCapacity
	}
	if valueCodec == nil {
		return nil, ErrNilValueCodec
	}
	if valueCodec.Size() == 0 {
		return nil, ErrInvalidValueSize
	}

	return &ORAMClient[V]{
		capacity: capacity,
		codec:    valueCodec,
		dataFile: dataFile,
	}, nil
}

// Default creates an eoram store using the default tmpfs data file.
func Default[V any](
	capacity uint64,
	valueCodec codec.ValueCodec[V],
) (*ORAMClient[V], error) {
	return New(capacity, valueCodec, DefaultDataFile)
}
