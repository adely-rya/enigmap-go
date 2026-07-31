package eoram

import (
	"fmt"

	"github.com/adely-rya/oram/core"
)

// LargeBucket is the fixed-size physical I/O unit containing a complete
// LevelsPerLargeBucket-level ORAM subtree.
type LargeBucket[T core.Selectable[T]] []Bucket[T]

func NewLargeBucket[T core.Selectable[T]](bucket Bucket[T]) LargeBucket[T] {
	largeBucket := make(LargeBucket[T], BucketsPerLargeBucket)
	for index := range largeBucket {
		largeBucket[index] = cloneBucket(bucket)
	}
	return largeBucket
}

func (largeBucket LargeBucket[T]) Serialize() ([]byte, error) {
	if len(largeBucket) != BucketsPerLargeBucket {
		return nil, fmt.Errorf(
			"eoram: large bucket must contain %d buckets: got=%d",
			BucketsPerLargeBucket,
			len(largeBucket),
		)
	}

	encoded := make([]byte, serializedLargeBucketSize)
	for index, bucket := range largeBucket {
		bucketBytes, err := bucket.Serialize()
		if err != nil {
			return nil, fmt.Errorf("eoram: serialize large bucket entry %d: %w", index, err)
		}
		start := index * serializedBucketSize
		copy(encoded[start:start+serializedBucketSize], bucketBytes)
	}
	return encoded, nil
}

func (LargeBucket[T]) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != serializedLargeBucketSize {
		return nil, fmt.Errorf(
			"eoram: serialized large bucket size mismatch: got=%d want=%d",
			len(encoded),
			serializedLargeBucketSize,
		)
	}

	largeBucket := make(LargeBucket[T], BucketsPerLargeBucket)
	for index := range largeBucket {
		start := index * serializedBucketSize
		decoded, err := (Bucket[T]{}).Deserialize(encoded[start : start+serializedBucketSize])
		if err != nil {
			return nil, fmt.Errorf("eoram: deserialize large bucket entry %d: %w", index, err)
		}
		largeBucket[index] = decoded.(Bucket[T])
	}
	return largeBucket, nil
}

func (LargeBucket[T]) ContentSize() uint64 { return serializedLargeBucketSize }
func (LargeBucket[T]) Size() uint64        { return serializedLargeBucketSize }

func cloneBucket[T core.Selectable[T]](bucket Bucket[T]) Bucket[T] {
	cloned := make(Bucket[T], len(bucket))
	copy(cloned, bucket)
	return cloned
}

func cloneLargeBucket[T core.Selectable[T]](largeBucket LargeBucket[T]) LargeBucket[T] {
	cloned := make(LargeBucket[T], len(largeBucket))
	for index, bucket := range largeBucket {
		cloned[index] = cloneBucket(bucket)
	}
	return cloned
}
