package eoram

import (
	"crypto/subtle"

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

func (block nodeBlock) Select(candidate nodeBlock, condition int) nodeBlock {
	condition = subtle.ConstantTimeEq(int32(condition), 1)

	currentKey := block.node.Key.Bytes()
	candidateKey := candidate.node.Key.Bytes()
	var selectedKey [avl.KeySize]byte
	for index := range selectedKey {
		selectedKey[index] = byte(subtle.ConstantTimeSelect(
			condition,
			int(candidateKey[index]),
			int(currentKey[index]),
		))
	}

	var selectedValue omap.Value
	for index := range selectedValue {
		selectedValue[index] = byte(subtle.ConstantTimeSelect(
			condition,
			int(candidate.node.Value[index]),
			int(block.node.Value[index]),
		))
	}

	return nodeBlock{
		node: avl.Node[omap.Value]{
			Key:     avl.NewKey(selectedKey[:]),
			Value:   selectedValue,
			Balance: avl.Balance(selectUint64(condition, uint64(block.node.Balance), uint64(candidate.node.Balance))),
			Left:    avl.NodeID(selectUint64(condition, uint64(block.node.Left), uint64(candidate.node.Left))),
			Right:   avl.NodeID(selectUint64(condition, uint64(block.node.Right), uint64(candidate.node.Right))),
			Valid:   uint8(subtle.ConstantTimeSelect(condition, int(candidate.node.Valid), int(block.node.Valid))),
		},
	}
}

func selectUint64(condition int, current, candidate uint64) uint64 {
	mask := uint64(0) - uint64(condition&1)
	return (current &^ mask) | (candidate & mask)
}
