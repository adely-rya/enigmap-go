package eoram

import (
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
)

// Block is the logical unit stored in an ORAM bucket.
type Block struct {
	Address  avl.NodeID
	Position avl.Position
	Node     avl.Node[omap.Value]
	Valid    uint8
}

// Bucket is one logical Path ORAM bucket.
type Bucket []Block
