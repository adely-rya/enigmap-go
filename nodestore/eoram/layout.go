package eoram

import (
	"fmt"
	"math/bits"
)

type LargeBucketLocation struct {
	PhysicalIndex uint64
	InnerIndex    int
}

type LargeBucketLayout struct {
	logicalBucketCount uint64
	physicalCount      uint64
}

func NewLargeBucketLayout(logicalBucketCount uint64) (LargeBucketLayout, error) {
	leafCount := logicalBucketCount/2 + 1
	if leafCount == 0 || leafCount&(leafCount-1) != 0 || 2*leafCount-1 != logicalBucketCount {
		return LargeBucketLayout{}, ErrInvalidTreeSize
	}

	levels := bits.Len64(leafCount)
	paddedLevels := ((levels + LevelsPerLargeBucket - 1) / LevelsPerLargeBucket) * LevelsPerLargeBucket
	paddedBucketCount := (uint64(1) << paddedLevels) - 1
	return LargeBucketLayout{
		logicalBucketCount: logicalBucketCount,
		physicalCount:      paddedBucketCount / BucketsPerLargeBucket,
	}, nil
}

func (layout LargeBucketLayout) LogicalBucketCount() uint64  { return layout.logicalBucketCount }
func (layout LargeBucketLayout) PhysicalBucketCount() uint64 { return layout.physicalCount }

func (layout LargeBucketLayout) Locate(logicalIndex uint64) (LargeBucketLocation, error) {
	if logicalIndex >= layout.logicalBucketCount {
		return LargeBucketLocation{}, fmt.Errorf(
			"eoram: logical bucket index out of range: index=%d size=%d",
			logicalIndex,
			layout.logicalBucketCount,
		)
	}

	depth := bits.Len64(logicalIndex+1) - 1
	levelStart := (uint64(1) << depth) - 1
	positionInLevel := logicalIndex - levelStart
	rootDepth := depth - depth%LevelsPerLargeBucket
	localDepth := depth - rootDepth
	rootPosition := positionInLevel >> localDepth
	physicalLevelStart := ((uint64(1) << rootDepth) - 1) / BucketsPerLargeBucket
	localPositionMask := (uint64(1) << localDepth) - 1
	innerIndex := ((uint64(1) << localDepth) - 1) + (positionInLevel & localPositionMask)

	return LargeBucketLocation{
		PhysicalIndex: physicalLevelStart + rootPosition,
		InnerIndex:    int(innerIndex),
	}, nil
}
