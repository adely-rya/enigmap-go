package enigmap

import (
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/memory"
	"github.com/adely-rya/enigmap-go/omap"
)

func NewMemory[V any](
	maxNodes uint64,
	selector omap.ValueSelector[V],
) (omap.Store[V], error) {
	store := memory.New[V]()
	return avl.NewTree[V](store, maxNodes, selector)
}
