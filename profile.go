package enigmap

import (
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/enigmap-go/padding"
)

// NewORAMForProfile exposes internal sizing parameters for diagnostic runs.
// Normal callers should use NewORAM or NewORAMWithCacheCapacity.
func NewORAMForProfile[T ORAMValue[T]](
	filePath string,
	maxNodes uint64,
	key []byte,
	cacheCapacity int,
	stashSize int,
) (*ORAMMap[T], error) {
	return newORAMWithParameters[T](filePath, maxNodes, key, cacheCapacity, stashSize)
}

// NewORAMForProfileUnencrypted bypasses AES-GCM solely to isolate encryption
// cost. It must not be used for security-sensitive or production workloads.
func NewORAMForProfileUnencrypted[T ORAMValue[T]](
	filePath string,
	maxNodes uint64,
	cacheCapacity int,
) (*ORAMMap[T], error) {
	requiredLeaves := maxNodes + 2
	if requiredLeaves < maxNodes {
		return nil, fmt.Errorf("enigmap: maxNodes is too large")
	}

	leafCount := uint64(1)
	for leafCount < requiredLeaves {
		if leafCount > ^uint64(0)/2 {
			return nil, fmt.Errorf("enigmap: ORAM tree size overflow")
		}
		leafCount *= 2
	}
	bucketCount := 2*leafCount - 1
	layout, err := eoram.NewLargeBucketLayout(bucketCount)
	if err != nil {
		return nil, err
	}

	type largeBucket = eoram.LargeBucket[eoram.NodeBlock[T]]
	file, err := padding.New[largeBucket](
		filePath,
		layout.PhysicalBucketCount(),
	)
	if err != nil {
		return nil, err
	}
	largeBucketStorage, err := eoram.NewLargeBucketStorage[eoram.NodeBlock[T]](
		file,
		layout,
		cacheCapacity,
	)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	client, err := eoram.NewOramClient[T](largeBucketStorage, eoram.BucketSize)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	tree, err := avl.NewTree[T](client, maxNodes)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &ORAMMap[T]{
		Tree:             tree,
		file:             file,
		largeBucketStore: largeBucketStorage,
		client:           client,
	}, nil
}

// EnableNonObliviousEvictionForProfile enables an unsafe experiment mode.
// The resulting CPU trace depends on data and must not be used in production.
func (store *ORAMMap[T]) EnableNonObliviousEvictionForProfile() {
	store.client.EnableNonObliviousEvictionForProfile()
}
