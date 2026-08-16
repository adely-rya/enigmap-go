package eoram

import (
	"fmt"

	"github.com/adely-rya/oram/core"
)

// LargeBucket is the fixed-size physical I/O unit containing a complete
// LevelsPerLargeBucket-level ORAM subtree.
type LargeBucket[T core.Selectable[T]] []Bucket[T]

func NewLargeBucket[T core.Selectable[T]](bucket Bucket[T]) LargeBucket[T] {
	largeBucket, blocks := makeLargeBucketBuffer[T](BucketsPerLargeBucket, len(bucket))
	for index := range largeBucket {
		start := index * len(bucket)
		largeBucket[index] = blocks[start : start+len(bucket)]
		copy(largeBucket[index], bucket)
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
	for bucketIndex, bucket := range largeBucket {
		if len(bucket) != serializedBucketBlockCount {
			return nil, fmt.Errorf(
				"eoram: large bucket entry %d must contain %d blocks: got=%d",
				bucketIndex,
				serializedBucketBlockCount,
				len(bucket),
			)
		}
		bucketStart := bucketIndex * serializedBucketSize
		for blockIndex := range bucket {
			blockStart := bucketStart + blockIndex*serializedBlockSize
			if err := encodeBlock(
				encoded[blockStart:blockStart+serializedBlockSize],
				bucket[blockIndex],
			); err != nil {
				return nil, fmt.Errorf(
					"eoram: serialize large bucket entry %d block %d: %w",
					bucketIndex,
					blockIndex,
					err,
				)
			}
		}
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

	largeBucket, blocks := makeLargeBucketBuffer[T](
		BucketsPerLargeBucket,
		serializedBucketBlockCount,
	)
	for bucketIndex := range largeBucket {
		blockBase := bucketIndex * serializedBucketBlockCount
		largeBucket[bucketIndex] = blocks[blockBase : blockBase+serializedBucketBlockCount]
		bucketStart := bucketIndex * serializedBucketSize
		for blockIndex := 0; blockIndex < serializedBucketBlockCount; blockIndex++ {
			blockStart := bucketStart + blockIndex*serializedBlockSize
			block, err := decodeBlock[T](encoded[blockStart : blockStart+serializedBlockSize])
			if err != nil {
				return nil, fmt.Errorf(
					"eoram: deserialize large bucket entry %d block %d: %w",
					bucketIndex,
					blockIndex,
					err,
				)
			}
			largeBucket[bucketIndex][blockIndex] = block
		}
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
	if len(largeBucket) == 0 {
		return nil
	}
	bucketSize := len(largeBucket[0])
	for _, bucket := range largeBucket[1:] {
		if len(bucket) != bucketSize {
			fallback := make(LargeBucket[T], len(largeBucket))
			for index, fallbackBucket := range largeBucket {
				fallback[index] = cloneBucket(fallbackBucket)
			}
			return fallback
		}
	}

	cloned, blocks := makeLargeBucketBuffer[T](len(largeBucket), bucketSize)
	for index, bucket := range largeBucket {
		start := index * bucketSize
		cloned[index] = blocks[start : start+bucketSize]
		copy(cloned[index], bucket)
	}
	return cloned
}

func makeLargeBucketBuffer[T core.Selectable[T]](
	bucketCount int,
	bucketSize int,
) (LargeBucket[T], []Block[T]) {
	return make(LargeBucket[T], bucketCount), make([]Block[T], bucketCount*bucketSize)
}

func copyBucketInto[T core.Selectable[T]](dst Bucket[T], src Bucket[T]) Bucket[T] {
	if len(dst) != len(src) {
		return cloneBucket(src)
	}
	copy(dst, src)
	return dst
}

func copyLargeBucketInto[T core.Selectable[T]](dst LargeBucket[T], src LargeBucket[T]) LargeBucket[T] {
	if len(dst) != len(src) {
		return cloneLargeBucket(src)
	}
	for index := range src {
		dst[index] = copyBucketInto(dst[index], src[index])
	}
	return dst
}

func cloneBucketPath[T core.Selectable[T]](path []Bucket[T]) []Bucket[T] {
	if len(path) == 0 {
		return nil
	}
	bucketSize := len(path[0])
	for _, bucket := range path[1:] {
		if len(bucket) != bucketSize {
			fallback := make([]Bucket[T], len(path))
			for index, fallbackBucket := range path {
				fallback[index] = cloneBucket(fallbackBucket)
			}
			return fallback
		}
	}

	cloned, blocks := makeBucketSliceBuffer[T](len(path), bucketSize)
	for index, bucket := range path {
		start := index * bucketSize
		cloned[index] = blocks[start : start+bucketSize]
		copy(cloned[index], bucket)
	}
	return cloned
}

func makeBucketSliceBuffer[T core.Selectable[T]](
	bucketCount int,
	bucketSize int,
) ([]Bucket[T], []Block[T]) {
	return make([]Bucket[T], bucketCount), make([]Block[T], bucketCount*bucketSize)
}
