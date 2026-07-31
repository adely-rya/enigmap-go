package eoram

import "testing"

func TestLargeBucketLayoutPacksCompleteSubtrees(t *testing.T) {
	logicalBucketCount := uint64((1 << (LevelsPerLargeBucket + 1)) - 1)
	layout, err := NewLargeBucketLayout(logicalBucketCount)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	paddedLevels := ((LevelsPerLargeBucket + 1 + LevelsPerLargeBucket - 1) /
		LevelsPerLargeBucket) * LevelsPerLargeBucket
	wantPhysicalCount := uint64((1<<paddedLevels)-1) / BucketsPerLargeBucket
	if layout.PhysicalBucketCount() != wantPhysicalCount {
		t.Fatalf("physical count = %d, want %d", layout.PhysicalBucketCount(), wantPhysicalCount)
	}

	seen := make(map[LargeBucketLocation]bool, logicalBucketCount)
	for logicalIndex := uint64(0); logicalIndex < logicalBucketCount; logicalIndex++ {
		got, err := layout.Locate(logicalIndex)
		if err != nil {
			t.Fatalf("Locate(%d): %v", logicalIndex, err)
		}
		if got.PhysicalIndex >= layout.PhysicalBucketCount() || got.InnerIndex >= BucketsPerLargeBucket {
			t.Fatalf("Locate(%d) returned out-of-range location %+v", logicalIndex, got)
		}
		if seen[got] {
			t.Fatalf("Locate(%d) returned duplicate location %+v", logicalIndex, got)
		}
		seen[got] = true
	}
}

func TestLargeBucketLayoutPadsIncompleteLevelGroup(t *testing.T) {
	layout, err := NewLargeBucketLayout(7)
	if err != nil {
		t.Fatalf("NewLargeBucketLayout: %v", err)
	}
	paddedLevels := ((3 + LevelsPerLargeBucket - 1) / LevelsPerLargeBucket) * LevelsPerLargeBucket
	want := uint64((1<<paddedLevels)-1) / BucketsPerLargeBucket
	if layout.PhysicalBucketCount() != want {
		t.Fatalf("physical count = %d, want %d", layout.PhysicalBucketCount(), want)
	}
}
