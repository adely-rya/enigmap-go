package eoram

import (
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/utils"
)

// BlockAddress identifies a logical ORAM block.
type BlockAddress uint64

// Leaf identifies a leaf in the ORAM tree.
type Leaf uint64

// NodeBlock is the selectable ORAM payload containing one complete AVL node.
type NodeBlock[V core.Selectable[V]] struct {
	Node avl.Node[V]
}

const (
	serializedBalanceOffset   = KeyValueSize
	serializedLeftOffset      = serializedBalanceOffset + 1
	serializedRightOffset     = serializedLeftOffset + 16
	serializedNodeValidOffset = serializedRightOffset + 16
)

// Serialize encodes the complete AVL node stored in one ORAM block.
func (block NodeBlock[V]) Serialize() ([]byte, error) {
	encoded := make([]byte, serializedNodeBlockSize)
	block.Node.Key.CopyBytes(encoded[:avl.KeySize])

	if fixed, ok := any(block.Node.Value).(FixedValue); ok {
		copy(encoded[avl.KeySize:KeyValueSize], fixed[:])
	} else {
		value, ok := any(block.Node.Value).(core.Serializable)
		if !ok {
			return nil, fmt.Errorf("eoram: node value does not implement core.Serializable")
		}
		valueBytes, err := value.Serialize()
		if err != nil {
			return nil, fmt.Errorf("eoram: serialize node value: %w", err)
		}
		if len(valueBytes) != ValueSize {
			return nil, fmt.Errorf(
				"eoram: serialized node value size mismatch: got=%d want=%d",
				len(valueBytes),
				ValueSize,
			)
		}
		copy(encoded[avl.KeySize:KeyValueSize], valueBytes)
	}

	encoded[serializedBalanceOffset] = byte(block.Node.Balance)
	encodePointer(encoded[serializedLeftOffset:serializedRightOffset], block.Node.Left)
	encodePointer(encoded[serializedRightOffset:serializedNodeValidOffset], block.Node.Right)
	encoded[serializedNodeValidOffset] = block.Node.Valid

	return encoded, nil
}

// Deserialize decodes one complete AVL node.
func (NodeBlock[V]) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != serializedNodeBlockSize {
		return nil, fmt.Errorf(
			"eoram: serialized node block size mismatch: got=%d want=%d",
			len(encoded),
			serializedNodeBlockSize,
		)
	}

	var zeroValue V
	valueDecoder, ok := any(zeroValue).(core.Serializable)
	if !ok {
		return nil, fmt.Errorf("eoram: node value does not implement core.Serializable")
	}
	decodedValue, err := valueDecoder.Deserialize(
		encoded[avl.KeySize:KeyValueSize],
	)
	if err != nil {
		return nil, fmt.Errorf("eoram: deserialize node value: %w", err)
	}
	value, ok := decodedValue.(V)
	if !ok {
		return nil, fmt.Errorf("eoram: deserialized node value has unexpected type")
	}

	return NodeBlock[V]{
		Node: avl.Node[V]{
			Key:     avl.NewKey(encoded[:avl.KeySize]),
			Value:   value,
			Balance: avl.Balance(encoded[serializedBalanceOffset]),
			Left:    decodePointer(encoded[serializedLeftOffset:serializedRightOffset]),
			Right:   decodePointer(encoded[serializedRightOffset:serializedNodeValidOffset]),
			Valid:   encoded[serializedNodeValidOffset],
		},
	}, nil
}

// ContentSize returns the fixed plaintext size of one node payload.
func (NodeBlock[V]) ContentSize() uint64 {
	return serializedNodeBlockSize
}

func encodePointer(encoded []byte, pointer avl.Pointer) {
	binary.LittleEndian.PutUint64(encoded[0:8], uint64(pointer.Address))
	binary.LittleEndian.PutUint64(encoded[8:16], uint64(pointer.Position))
}

func decodePointer(encoded []byte) avl.Pointer {
	return avl.Pointer{
		Address:  avl.NodeID(binary.LittleEndian.Uint64(encoded[0:8])),
		Position: avl.ORAMPosition(binary.LittleEndian.Uint64(encoded[8:16])),
	}
}

func (block NodeBlock[V]) Select(
	candidate NodeBlock[V],
	condition int,
) NodeBlock[V] {
	condition &= 1

	return NodeBlock[V]{
		Node: avl.Node[V]{
			Key:     avl.SelectKey(block.Node.Key, candidate.Node.Key, condition),
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
	BlockAddress BlockAddress
	Leaf         Leaf
	Data         T
	Valid        int
}

func (block Block[T]) Select(
	candidate Block[T],
	condition int,
) Block[T] {
	condition &= 1
	return Block[T]{
		BlockAddress: BlockAddress(utils.CtSelect64(
			condition,
			uint64(candidate.BlockAddress),
			uint64(block.BlockAddress),
		)),
		Leaf: Leaf(utils.CtSelect64(
			condition,
			uint64(candidate.Leaf),
			uint64(block.Leaf),
		)),
		Data: core.Select(block.Data, candidate.Data, condition),
		Valid: subtle.ConstantTimeSelect(
			condition,
			candidate.Valid,
			block.Valid,
		),
	}
}
