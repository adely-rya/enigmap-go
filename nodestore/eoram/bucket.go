package eoram

import (
	"encoding/binary"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/core"
)

type Bucket[T core.Selectable[T]] []Block[T]

const (
	// KeyValueSize is the fixed serialized size of an AVL key and value.
	// Change this constant to tune the ORAM payload size. ValueSize is derived
	// from it so callers do not need to duplicate the value length.
	KeyValueSize = 34
	ValueSize    = KeyValueSize - avl.KeySize

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
	DefaultLargeBucketCacheCapacity = 85
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
