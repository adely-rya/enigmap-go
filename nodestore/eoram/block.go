package eoram

import "github.com/adely-rya/enigmap-go/avl"

// Address identifies a logical ORAM block.
type Address uint64

// Position identifies a leaf in the ORAM tree.
type Position uint64

// Block is the logical Path ORAM block used by the AVL NodeStore.
type Block[V any] struct {
	Address  Address
	Position Position
	Data     avl.Node[V]
	Valid    int
}
