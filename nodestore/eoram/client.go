package eoram

import (
	"errors"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
)

var (
	ErrInvalidCapacity = errors.New("eoram: capacity must be positive")
	ErrNotImplemented  = errors.New("eoram: Path ORAM storage is not implemented")
)

const DefaultDataFile = "/dev/shm/enigmap/oram-large-buckets.db"

// ORAMClient is the ENIGMAP-specific AVL node store backed by Path ORAM.
//
// The constructor owns the stable configuration boundary. Path ORAM, Large
// Bucket cache, encryption, and file storage are added behind this type.
type ORAMClient struct {
	capacity uint64
	dataFile string
}

// New creates the ENIGMAP-specific Path ORAM node store configuration.
func New(
	capacity uint64,
	dataFile string,
) (*ORAMClient, error) {
	if capacity == 0 {
		return nil, ErrInvalidCapacity
	}

	return &ORAMClient{
		capacity: capacity,
		dataFile: dataFile,
	}, nil
}

// Default creates an eoram store using the default tmpfs data file.
func Default(
	capacity uint64,
) (*ORAMClient, error) {
	return New(capacity, DefaultDataFile)
}

// Allocate reserves a logical ORAM address and stores node at that address.
func (client *ORAMClient) Allocate(
	node avl.Node[omap.Value],
) (avl.NodeID, error) {
	_ = newNodeBlock(node)
	return avl.NullAddress, ErrNotImplemented
}

// Load reads one complete AVL node through Path ORAM.
func (client *ORAMClient) Load(
	id avl.NodeID,
) (avl.Node[omap.Value], error) {
	_ = id
	return avl.Node[omap.Value]{}, ErrNotImplemented
}

// Update replaces one complete AVL node through Path ORAM.
func (client *ORAMClient) Update(
	id avl.NodeID,
	node avl.Node[omap.Value],
) error {
	_, _ = id, newNodeBlock(node)
	return ErrNotImplemented
}

// FreeMaybe releases id without making the physical access trace depend on
// condition.
func (client *ORAMClient) FreeMaybe(
	id avl.NodeID,
	condition int,
) error {
	_, _ = id, condition
	return ErrNotImplemented
}

var _ avl.NodeStore[omap.Value] = (*ORAMClient)(nil)
