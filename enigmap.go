package enigmap

import (
	"errors"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/enigmap-go/nodestore/memory"
	"github.com/adely-rya/enigmap-go/omap"
	"github.com/adely-rya/oram/core"
)

var ErrORAMNodeStoreNotImplemented = errors.New(
	"enigmap: ORAM NodeStore is not implemented",
)

func NewMemory[V any](
	maxNodes uint64,
) (omap.Store[V], error) {
	store := memory.New[V]()
	return avl.NewTree[V](store, maxNodes)
}

// NewORAM prepares an ORAM-backed map.
//
// The final AVL connection remains disabled until the eoram NodeStore adapter
// is implemented.
func NewORAM[T core.Selectable[T]](
	storage core.IO[eoram.Bucket[T]],
	maxNodes uint64,
	bucketSize int,
) (omap.Store[T], error) {
	client, err := eoram.NewOramClient(storage, bucketSize)
	if err != nil {
		return nil, err
	}

	// nodeStore := eoram.NewStore(client)
	// return avl.NewTree[T](nodeStore, maxNodes)
	_ = client
	_ = maxNodes

	return nil, ErrORAMNodeStoreNotImplemented
}
