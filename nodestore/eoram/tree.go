package eoram

import (
	"fmt"
	"math/bits"

	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/oram/pathoram"
)

const reservedNodeCount = 2

type treeParameters struct {
	logicalBlockCount uint64
	leafCount         uint64
	bucketCount       uint64
	height            int
}

func newTreeParameters(capacity uint64) (treeParameters, error) {
	logicalBlockCount := capacity + reservedNodeCount
	if logicalBlockCount < capacity {
		return treeParameters{}, fmt.Errorf("eoram: capacity overflows logical block count")
	}

	leafCount := uint64(1)
	for leafCount < logicalBlockCount {
		if leafCount > ^uint64(0)/2 {
			return treeParameters{}, fmt.Errorf("eoram: capacity overflows leaf count")
		}
		leafCount <<= 1
	}

	return treeParameters{
		logicalBlockCount: logicalBlockCount,
		leafCount:         leafCount,
		bucketCount:       2*leafCount - 1,
		height:            bits.Len64(leafCount) - 1,
	}, nil
}

func initializeTree(
	store core.IO[pathoram.Bucket[nodeBlock]],
) error {
	empty := make(pathoram.Bucket[nodeBlock], pathoram.BUCKET_SIZE)
	for index := uint64(0); index < store.Size(); index++ {
		if err := store.Write(empty, index); err != nil {
			return fmt.Errorf("eoram: initialize bucket %d: %w", index, err)
		}
	}
	return nil
}
