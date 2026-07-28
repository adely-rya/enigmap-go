package enigmap

import (
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/memory"
	"github.com/adely-rya/enigmap-go/omap"
)

func NewMemory[V any]() omap.Store[V] {
	store := memory.New[V]()
	return avl.NewTree[V](store)
}
