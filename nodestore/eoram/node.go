package eoram

import (
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
)

// nodeBlock is the fixed-size AVL node representation stored in Path ORAM.
// Serialization and constant-time selection are added with the Path ORAM
// storage implementation.
type nodeBlock struct {
	node avl.Node[omap.Value]
}

func newNodeBlock(node avl.Node[omap.Value]) nodeBlock {
	return nodeBlock{node: node}
}

func (block nodeBlock) avlNode() avl.Node[omap.Value] {
	return block.node
}
