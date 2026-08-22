package eoram

import (
	"encoding/binary"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/config"
	"github.com/adely-rya/oram/core"
)

type Bucket[T core.Selectable[T]] []Block[T]

const (
	// These aliases preserve the eoram API while package config remains the
	// single source of truth for fixed key and value lengths.
	KeyValueSize = config.KeyValueSize
	ValueSize    = config.ValueSize

	// balance (1), two child addresses (16), two child leaves (16), valid (1).
	serializedAVLMetadataSize = 34
	serializedNodeBlockSize   = KeyValueSize + serializedAVLMetadataSize

	serializedBlockHeaderSize = 24
	serializedBlockSize       = serializedBlockHeaderSize + serializedNodeBlockSize

	BucketSize                 = 5
	serializedBucketBlockCount = BucketSize
	serializedBucketSize       = serializedBucketBlockCount * serializedBlockSize

	// LevelsPerLargeBucket controls how many consecutive ORAM tree levels are
	// packed into one physical file value.
	LevelsPerLargeBucket            = 3
	BucketsPerLargeBucket           = (1 << LevelsPerLargeBucket) - 1
	serializedLargeBucketSize       = BucketsPerLargeBucket * serializedBucketSize
	DefaultLargeBucketCacheCapacity = 50
)

// Serialize encodes a fixed-size bucket for encryption or file storage.
func (bucket Bucket[T]) Serialize() ([]byte, error) {
	if len(bucket) != serializedBucketBlockCount {
		return nil, fmt.Errorf(
			"eoram: bucket must contain %d blocks: got=%d",
			serializedBucketBlockCount,
			len(bucket),
		)
	}

	encoded := make([]byte, serializedBucketSize)
	for index := range bucket {
		start := index * serializedBlockSize
		end := start + serializedBlockSize
		if err := encodeBlock(encoded[start:end], bucket[index]); err != nil {
			return nil, fmt.Errorf("eoram: encode bucket block %d: %w", index, err)
		}
	}

	return encoded, nil
}

// Deserialize decodes one fixed-size bucket.
func (Bucket[T]) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != serializedBucketSize {
		return nil, fmt.Errorf(
			"eoram: serialized bucket size mismatch: got=%d want=%d",
			len(encoded),
			serializedBucketSize,
		)
	}

	bucket := make(Bucket[T], serializedBucketBlockCount)
	for index := range bucket {
		start := index * serializedBlockSize
		end := start + serializedBlockSize

		block, err := decodeBlock[T](encoded[start:end])
		if err != nil {
			return nil, fmt.Errorf("eoram: decode bucket block %d: %w", index, err)
		}
		bucket[index] = block
	}

	return bucket, nil
}

// ContentSize returns the fixed plaintext size of one serialized bucket.
func (Bucket[T]) ContentSize() uint64 {
	return serializedBucketSize
}

// Size returns the fixed encoded size required by plain FileStorage.
func (Bucket[T]) Size() uint64 {
	return serializedBucketSize
}

func encodeBlock[T core.Selectable[T]](
	encoded []byte,
	block Block[T],
) error {
	if len(encoded) != serializedBlockSize {
		return fmt.Errorf(
			"serialized block size mismatch: got=%d want=%d",
			len(encoded),
			serializedBlockSize,
		)
	}

	if fixed, ok := any(block).(Block[NodeBlock[FixedValue]]); ok {
		encodeFixedValueBlock(encoded, fixed)
		return nil
	}

	payload, ok := any(block.Data).(core.Serializable)
	if !ok {
		return fmt.Errorf("block payload does not implement core.Serializable")
	}
	payloadBytes, err := payload.Serialize()
	if err != nil {
		return fmt.Errorf("serialize block payload: %w", err)
	}
	if len(payloadBytes) != serializedNodeBlockSize {
		return fmt.Errorf(
			"serialized block payload size mismatch: got=%d want=%d",
			len(payloadBytes),
			serializedNodeBlockSize,
		)
	}

	binary.LittleEndian.PutUint64(encoded[0:8], uint64(block.BlockAddress))
	binary.LittleEndian.PutUint64(encoded[8:16], uint64(block.Leaf))
	binary.LittleEndian.PutUint64(encoded[16:24], uint64(int64(block.Valid)))
	copy(encoded[serializedBlockHeaderSize:], payloadBytes)

	return nil
}

func decodeBlock[T core.Selectable[T]](encoded []byte) (Block[T], error) {
	var zeroBlock Block[T]
	if len(encoded) != serializedBlockSize {
		return zeroBlock, fmt.Errorf(
			"serialized block size mismatch: got=%d want=%d",
			len(encoded),
			serializedBlockSize,
		)
	}

	var zeroPayload T
	if _, ok := any(zeroPayload).(NodeBlock[FixedValue]); ok {
		fixed, err := decodeFixedValueBlock(encoded)
		if err != nil {
			return zeroBlock, err
		}
		return any(fixed).(Block[T]), nil
	}

	payload, ok := any(zeroPayload).(core.Serializable)
	if !ok {
		return zeroBlock, fmt.Errorf("block payload does not implement core.Serializable")
	}
	decoded, err := payload.Deserialize(encoded[serializedBlockHeaderSize:])
	if err != nil {
		return zeroBlock, fmt.Errorf("deserialize block payload: %w", err)
	}
	data, ok := decoded.(T)
	if !ok {
		return zeroBlock, fmt.Errorf("deserialized block payload has unexpected type")
	}

	return Block[T]{
		BlockAddress: BlockAddress(binary.LittleEndian.Uint64(encoded[0:8])),
		Leaf:         Leaf(binary.LittleEndian.Uint64(encoded[8:16])),
		Valid:        int(int64(binary.LittleEndian.Uint64(encoded[16:24]))),
		Data:         data,
	}, nil
}

func encodeFixedValueBlock(
	encoded []byte,
	block Block[NodeBlock[FixedValue]],
) {
	binary.LittleEndian.PutUint64(encoded[0:8], uint64(block.BlockAddress))
	binary.LittleEndian.PutUint64(encoded[8:16], uint64(block.Leaf))
	binary.LittleEndian.PutUint64(encoded[16:24], uint64(int64(block.Valid)))

	nodeEncoded := encoded[serializedBlockHeaderSize:]
	node := block.Data.Node
	node.Key.CopyBytes(nodeEncoded[:avl.KeySize])
	copy(nodeEncoded[avl.KeySize:KeyValueSize], node.Value[:])
	nodeEncoded[serializedBalanceOffset] = byte(node.Balance)
	encodePointer(nodeEncoded[serializedLeftOffset:serializedRightOffset], node.Left)
	encodePointer(nodeEncoded[serializedRightOffset:serializedNodeValidOffset], node.Right)
	nodeEncoded[serializedNodeValidOffset] = node.Valid
}

func decodeFixedValueBlock(
	encoded []byte,
) (Block[NodeBlock[FixedValue]], error) {
	var zero Block[NodeBlock[FixedValue]]
	if len(encoded) != serializedBlockSize {
		return zero, fmt.Errorf(
			"serialized block size mismatch: got=%d want=%d",
			len(encoded),
			serializedBlockSize,
		)
	}

	nodeEncoded := encoded[serializedBlockHeaderSize:]
	var value FixedValue
	copy(value[:], nodeEncoded[avl.KeySize:KeyValueSize])
	return Block[NodeBlock[FixedValue]]{
		BlockAddress: BlockAddress(binary.LittleEndian.Uint64(encoded[0:8])),
		Leaf:         Leaf(binary.LittleEndian.Uint64(encoded[8:16])),
		Valid:        int(int64(binary.LittleEndian.Uint64(encoded[16:24]))),
		Data: NodeBlock[FixedValue]{
			Node: avl.Node[FixedValue]{
				Key:     avl.NewKey(nodeEncoded[:avl.KeySize]),
				Value:   value,
				Balance: avl.Balance(nodeEncoded[serializedBalanceOffset]),
				Left:    decodePointer(nodeEncoded[serializedLeftOffset:serializedRightOffset]),
				Right:   decodePointer(nodeEncoded[serializedRightOffset:serializedNodeValidOffset]),
				Valid:   nodeEncoded[serializedNodeValidOffset],
			},
		},
	}, nil
}
