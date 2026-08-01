package eoram

import "testing"

func TestLargeBucketSerializationRoundTrip(t *testing.T) {
	empty := make(Bucket[serializablePayload], BucketSize)
	largeBucket := NewLargeBucket(empty)
	slot := BucketSize - 1
	largeBucket[2][slot] = Block[serializablePayload]{
		BlockAddress: 17,
		Leaf:         9,
		Data:         serializablePayload{0xaa, 0xbb},
		Valid:        1,
	}

	encoded, err := largeBucket.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if len(encoded) != serializedLargeBucketSize {
		t.Fatalf("serialized size = %d, want %d", len(encoded), serializedLargeBucketSize)
	}

	decodedValue, err := largeBucket.Deserialize(encoded)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	decoded := decodedValue.(LargeBucket[serializablePayload])
	if decoded[2][slot] != largeBucket[2][slot] {
		t.Fatalf("decoded block = %+v, want %+v", decoded[2][slot], largeBucket[2][slot])
	}
}

func TestLargeBucketContentSize(t *testing.T) {
	var largeBucket LargeBucket[serializablePayload]
	if got := largeBucket.ContentSize(); got != serializedLargeBucketSize {
		t.Fatalf("ContentSize = %d, want %d", got, serializedLargeBucketSize)
	}
}
