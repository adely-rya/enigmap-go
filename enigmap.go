package enigmap

import (
	"errors"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/enigmap-go/nodestore/memory"
	"github.com/adely-rya/enigmap-go/omap"
	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/security/encrypt"
	"github.com/adely-rya/oram/storage"
)

func NewMemory[V any](
	maxNodes uint64,
) (omap.Store[V], error) {
	store := memory.New[V]()
	return avl.NewTree[V](store, maxNodes)
}

type ORAMValue[T any] interface {
	core.Selectable[T]
	core.Serializable
}

type ORAMMap[T ORAMValue[T]] struct {
	*avl.Tree[T]
	file             *storage.FileStorage[encrypt.Ciphertext[eoram.LargeBucket[eoram.NodeBlock[T]]]]
	largeBucketStore *eoram.LargeBucketStorage[eoram.NodeBlock[T]]
}

func (store *ORAMMap[T]) Close() error {
	flushErr := store.largeBucketStore.Flush()
	closeErr := store.file.Close()
	return errors.Join(flushErr, closeErr)
}

// NewORAM creates an AES-GCM-encrypted, file-backed ORAM map.
func NewORAM[T ORAMValue[T]](
	filePath string,
	maxNodes uint64,
	key []byte,
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
	type ciphertext = encrypt.Ciphertext[largeBucket]

	file, err := storage.NewFileStorage[ciphertext](
		filePath,
		layout.PhysicalBucketCount(),
	)
	if err != nil {
		return nil, err
	}

	encryption, err := encrypt.NewEncryption[largeBucket, ciphertext](key)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("enigmap: initialize Large Bucket encryption: %w", err)
	}
	encryptedStorage := encrypt.NewEncryptIO(file, encryption)

	largeBucketStorage, err := eoram.NewLargeBucketStorage[eoram.NodeBlock[T]](
		encryptedStorage,
		layout,
		eoram.DefaultLargeBucketCacheCapacity,
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
	}, nil
}
