package eoram

import (
	"crypto/subtle"
	"testing"

	"github.com/adely-rya/oram/core"
)

type serializablePayload [serializedNodeBlockSize]byte

func (payload serializablePayload) Select(
	candidate serializablePayload,
	condition int,
) serializablePayload {
	var selected serializablePayload
	for index := range selected {
		selected[index] = byte(subtle.ConstantTimeSelect(
			condition&1,
			int(candidate[index]),
			int(payload[index]),
		))
	}
	return selected
}

func (payload serializablePayload) Serialize() ([]byte, error) {
	encoded := make([]byte, len(payload))
	copy(encoded, payload[:])
	return encoded, nil
}

func (serializablePayload) Deserialize(encoded []byte) (core.Serializable, error) {
	var payload serializablePayload
	copy(payload[:], encoded)
	return payload, nil
}

func TestBucketSerializationRoundTrip(t *testing.T) {
	bucket := make(Bucket[serializablePayload], serializedBucketBlockCount)
	bucket[2] = Block[serializablePayload]{
		BlockAddress: 42,
		Leaf:         3,
		Data:         serializablePayload{0xaa, 0xbb},
		Valid:        1,
	}

	encoded, err := bucket.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(encoded) != serializedBucketSize {
		t.Fatalf("serialized size = %d, want %d", len(encoded), serializedBucketSize)
	}

	decodedValue, err := bucket.Deserialize(encoded)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	decoded := decodedValue.(Bucket[serializablePayload])
	for index := range bucket {
		if decoded[index] != bucket[index] {
			t.Fatalf("decoded block %d = %+v, want %+v", index, decoded[index], bucket[index])
		}
	}
}

func TestBucketContentSize(t *testing.T) {
	var bucket Bucket[serializablePayload]
	if got := bucket.ContentSize(); got != serializedBucketSize {
		t.Fatalf("ContentSize = %d, want %d", got, serializedBucketSize)
	}
}
