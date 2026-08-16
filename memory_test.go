//go:build latency

package enigmap_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/oram/storage"
)

const oramMemoryBenchmarkWorkingSetSize = 10

// BenchmarkORAMMemoryStoreLatency measures the same ENIGMAP AVL + Path ORAM
// stack as the file-backed benchmark, but stores physical ORAM buckets directly
// in storage.MemoryStorage. AES-GCM and filesystem I/O are intentionally excluded.
func BenchmarkORAMMemoryStoreLatency(b *testing.B) {
	for _, benchmarkSize := range omapBenchmarkSizes {
		b.Run(benchmarkSize.name, func(b *testing.B) {
			benchmarkORAMMemoryStoreLatency(b, benchmarkSize.size)
		})
	}
}

func benchmarkORAMMemoryStoreLatency(b *testing.B, omapSize int) {
	b.Run("Initialization", func(b *testing.B) {
		b.ReportAllocs()

		for range b.N {
			store := newMemoryBackedORAM(b, omapSize)
			runtime.KeepAlive(store)
		}
	})

	b.Run("Put", func(b *testing.B) {
		store := newMemoryBackedORAM(b, omapSize)
		keys := makeORAMBenchmarkKeys(omapSize)
		values := makeORAMBenchmarkValues("data", omapSize)

		b.ReportAllocs()
		b.ResetTimer()

		for index := range b.N {
			b.StopTimer()
			if index > 0 && index%omapSize == 0 {
				store = newMemoryBackedORAM(b, omapSize)
			}
			b.StartTimer()

			keyIndex := index % omapSize
			if err := store.Put(keys[keyIndex], values[keyIndex]); err != nil {
				b.Fatalf("Put(%d): %v", keyIndex, err)
			}
		}
	})

	b.Run("OverwritePut", func(b *testing.B) {
		store := newPopulatedMemoryBackedORAM(
			b,
			omapSize,
			oramMemoryBenchmarkWorkingSetSize,
		)
		keys := makeORAMBenchmarkKeys(oramMemoryBenchmarkWorkingSetSize)
		values := makeORAMBenchmarkValues("updated", oramMemoryBenchmarkWorkingSetSize)

		b.ReportAllocs()
		b.ResetTimer()

		for index := range b.N {
			keyIndex := index % oramMemoryBenchmarkWorkingSetSize
			if err := store.Put(keys[keyIndex], values[keyIndex]); err != nil {
				b.Fatalf("overwrite Put(%d): %v", keyIndex, err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		store := newPopulatedMemoryBackedORAM(
			b,
			omapSize,
			oramMemoryBenchmarkWorkingSetSize,
		)
		keys := makeORAMBenchmarkKeys(oramMemoryBenchmarkWorkingSetSize)

		b.ReportAllocs()
		b.ResetTimer()

		for index := range b.N {
			keyIndex := index % oramMemoryBenchmarkWorkingSetSize
			_, found, err := store.Get(keys[keyIndex])
			if err != nil {
				b.Fatalf("Get(%d): %v", keyIndex, err)
			}
			if !found {
				b.Fatalf("Get(%d): key not found", keyIndex)
			}
		}
	})
}

func newMemoryBackedORAM(
	b *testing.B,
	maxNodes int,
) *avl.Tree[oramLatencyValue] {
	b.Helper()

	requiredLeaves := uint64(maxNodes) + 2
	leafCount := uint64(1)
	for leafCount < requiredLeaves {
		leafCount *= 2
	}
	bucketCount := 2*leafCount - 1

	layout, err := eoram.NewLargeBucketLayout(bucketCount)
	if err != nil {
		b.Fatalf("NewLargeBucketLayout: %v", err)
	}

	type largeBucket = eoram.LargeBucket[eoram.NodeBlock[oramLatencyValue]]

	memoryStorage := storage.NewMemoryStorage[largeBucket](
		layout.PhysicalBucketCount(),
	)

	largeBucketStorage, err := eoram.NewLargeBucketStorage[eoram.NodeBlock[oramLatencyValue]](
		memoryStorage,
		layout,
		eoram.DefaultLargeBucketCacheCapacity,
	)
	if err != nil {
		b.Fatalf("NewLargeBucketStorage: %v", err)
	}

	client, err := eoram.NewOramClient[oramLatencyValue](
		largeBucketStorage,
		eoram.BucketSize,
	)
	if err != nil {
		b.Fatalf("NewOramClient: %v", err)
	}

	tree, err := avl.NewTree[oramLatencyValue](client, uint64(maxNodes))
	if err != nil {
		b.Fatalf("NewTree: %v", err)
	}
	return tree
}

func newPopulatedMemoryBackedORAM(
	b *testing.B,
	maxNodes int,
	entryCount int,
) *avl.Tree[oramLatencyValue] {
	b.Helper()

	store := newMemoryBackedORAM(b, maxNodes)
	for index := 0; index < entryCount; index++ {
		if err := store.Put(
			oramLatencyKey(index),
			oramLatencyData("data", index),
		); err != nil {
			b.Fatalf("setup Put(%d): %v", index, err)
		}
	}
	return store
}

func makeORAMBenchmarkKeys(count int) [][]byte {
	keys := make([][]byte, count)
	for index := range count {
		keys[index] = oramLatencyKey(index)
	}
	return keys
}

func makeORAMBenchmarkValues(prefix string, count int) []oramLatencyValue {
	values := make([]oramLatencyValue, count)
	for index := range count {
		values[index] = oramLatencyData(prefix, index)
	}
	return values
}

func oramLatencyKey(index int) []byte {
	key := make([]byte, avl.KeySize)
	copy(key, fmt.Sprintf("key-%08d", index))
	return key
}

func oramLatencyData(prefix string, index int) oramLatencyValue {
	var value oramLatencyValue
	copy(value[:], fmt.Sprintf("%s-%08d", prefix, index))
	return value
}
