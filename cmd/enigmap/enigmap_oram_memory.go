package main

import (
	"crypto/rand"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/oram/security/encrypt"
	"github.com/adely-rya/oram/storage"
)

func runORAMMemoryEncrypted() error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate ORAM encryption key: %w", err)
	}

	store, err := newORAMMemoryEncrypted(maxNodes, key)
	if err != nil {
		return fmt.Errorf("create encrypted memory ORAM: %w", err)
	}
	if err := runExample(store); err != nil {
		return fmt.Errorf("run encrypted memory ORAM: %w", err)
	}

	fmt.Printf("oram/memory/aes-gcm: Put/overwrite/Get succeeded for all %d entries\n", entryCount)
	return nil
}

func newORAMMemoryEncrypted(
	maxNodes uint64,
	key []byte,
) (*avl.Tree[exampleValue], error) {
	requiredLeaves := maxNodes + 2
	if requiredLeaves < maxNodes {
		return nil, fmt.Errorf("maxNodes is too large")
	}

	leafCount := uint64(1)
	for leafCount < requiredLeaves {
		if leafCount > ^uint64(0)/2 {
			return nil, fmt.Errorf("ORAM tree size overflow")
		}
		leafCount *= 2
	}
	bucketCount := 2*leafCount - 1

	layout, err := eoram.NewLargeBucketLayout(bucketCount)
	if err != nil {
		return nil, err
	}

	type largeBucket = eoram.LargeBucket[eoram.NodeBlock[exampleValue]]
	type ciphertext = encrypt.Ciphertext[largeBucket]

	memoryStorage := storage.NewMemoryStorage[ciphertext](
		layout.PhysicalBucketCount(),
	)
	encryption, err := encrypt.NewEncryption[largeBucket, ciphertext](key)
	if err != nil {
		return nil, fmt.Errorf("initialize Large Bucket encryption: %w", err)
	}
	encryptedStorage := encrypt.NewEncryptIO(memoryStorage, encryption)

	largeBucketStorage, err := eoram.NewLargeBucketStorage[eoram.NodeBlock[exampleValue]](
		encryptedStorage,
		layout,
		eoram.DefaultLargeBucketCacheCapacity,
	)
	if err != nil {
		return nil, err
	}

	client, err := eoram.NewOramClient[exampleValue](
		largeBucketStorage,
		eoram.BucketSize,
	)
	if err != nil {
		return nil, err
	}
	return avl.NewTree[exampleValue](client, maxNodes)
}
