package eoram

import (
	"errors"

	"github.com/adely-rya/enigmap-go/codec"
)

var (
	ErrInvalidMaxNodes      = errors.New("eoram: maxNodes must be positive")
	ErrNilValueCodec        = errors.New("eoram: value codec is required")
	ErrInvalidValueSize     = errors.New("eoram: value codec size must be positive")
	ErrInvalidLevelsPerPack = errors.New("eoram: levels per pack must be positive")
	ErrInvalidCacheCapacity = errors.New("eoram: cache capacity must be positive")
)

// Store is the ENIGMAP-specific AVL node store backed by Path ORAM.
//
// The constructor owns the stable configuration boundary. Path ORAM, Large
// Bucket cache, encryption, and file storage are added behind this type.
type Store[V any] struct {
	maxNodes uint64
	codec    codec.ValueCodec[V]
	config   config
}

// New creates the ENIGMAP-specific Path ORAM node store configuration.
func New[V any](
	maxNodes uint64,
	valueCodec codec.ValueCodec[V],
	options ...Option,
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

	config := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	if config.levelsPerPack == 0 {
		return nil, ErrInvalidLevelsPerPack
	}
	if config.cacheCapacity <= 0 {
		return nil, ErrInvalidCacheCapacity
	}

	return &Store[V]{
		maxNodes: maxNodes,
		codec:    valueCodec,
		config:   config,
	}, nil
}
