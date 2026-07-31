package eoram

import (
	"crypto/subtle"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/utils"
)

// Address identifies a logical ORAM block.
type Address uint64

// Position identifies a leaf in the ORAM tree.
type Position uint64

// NodeBlock is the selectable ORAM payload containing one complete AVL node.
type NodeBlock[V core.Selectable[V]] struct {
	Node avl.Node[V]
}

func (block NodeBlock[V]) Select(
	candidate NodeBlock[V],
	condition int,
) NodeBlock[V] {
	condition &= 1

	currentKey := block.Node.Key.Bytes()
	candidateKey := candidate.Node.Key.Bytes()
	selectedKey := make([]byte, avl.KeySize)
	for index := 0; index < avl.KeySize; index++ {
		selectedKey[index] = byte(subtle.ConstantTimeSelect(
			condition,
			int(candidateKey[index]),
			int(currentKey[index]),
		))
	}

	return NodeBlock[V]{
		Node: avl.Node[V]{
			Key:     avl.NewKey(selectedKey),
			Value:   core.Select(block.Node.Value, candidate.Node.Value, condition),
			Balance: avl.Balance(subtle.ConstantTimeSelect(condition, int(candidate.Node.Balance), int(block.Node.Balance))),
			Left:    selectPointer(block.Node.Left, candidate.Node.Left, condition),
			Right:   selectPointer(block.Node.Right, candidate.Node.Right, condition),
			Valid:   uint8(subtle.ConstantTimeSelect(condition, int(candidate.Node.Valid), int(block.Node.Valid))),
		},
	}
}

func selectPointer(current, candidate avl.Pointer, condition int) avl.Pointer {
	return avl.Pointer{
		Address: avl.NodeID(utils.CtSelect64(
			condition,
			uint64(candidate.Address),
			uint64(current.Address),
		)),
		Position: avl.ORAMPosition(utils.CtSelect64(
			condition,
			uint64(candidate.Position),
			uint64(current.Position),
		)),
	}
}

// Block is a logical Path ORAM block.
type Block[T core.Selectable[T]] struct {
	Address  Address
	Position Position
	Data     T
	Valid    int
}

func (block Block[T]) Select(
	candidate Block[T],
	condition int,
) Block[T] {
	condition &= 1
	return Block[T]{
		Address: Address(utils.CtSelect64(
			condition,
			uint64(candidate.Address),
			uint64(block.Address),
		)),
		Position: Position(utils.CtSelect64(
			condition,
			uint64(candidate.Position),
			uint64(block.Position),
		)),
		Data: core.Select(block.Data, candidate.Data, condition),
		Valid: subtle.ConstantTimeSelect(
			condition,
			candidate.Valid,
			block.Valid,
		),
	}
}

type Bucket[T core.Selectable[T]] []Block[T]
