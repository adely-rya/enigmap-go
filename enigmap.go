package enigmap

import (
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/memory"
	"github.com/adely-rya/enigmap-go/omap"
)

func NewMemory[V any](
	capacity uint64,
) (omap.Store[V], error) {
	store := memory.New[V]()
	return avl.NewTree[V](store, capacity)
}

// NewORAM creates an oblivious map backed by the ENIGMAP Path ORAM node store.
//
// The factory will be enabled after eoram.ORAMClient implements
// avl.NodeStore[V].
//
// func NewORAM(
// 	capacity uint64,
// ) (omap.Store[omap.Value], error) {
// 	client, err := eoram.Default(capacity)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return avl.NewTree[omap.Value](client, capacity)
// }
