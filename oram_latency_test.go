//go:build latency

package enigmap_test

import (
	"fmt"
	"path/filepath"
	"testing"

	enigmap "github.com/adely-rya/enigmap-go"
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
)

const omapSize = 100_000_000

type oramLatencyValue = eoram.FixedValue

func BenchmarkORAMFileStoreLatency(b *testing.B) {
	// b.Run("Initialization", func(b *testing.B) {
	for index := range b.N {
		filePath := filepath.Join(
			b.TempDir(),
			fmt.Sprintf("oram-latency-%d.db", index),
		)

		store, err := enigmap.NewORAM[oramLatencyValue](
			filePath,
			omapSize,
			[]byte("0123456789abcdef0123456789abcdef"),
		)
		if err != nil {
			b.Fatalf("NewORAM: %v", err)
		}

		b.StopTimer()
		if err := store.Close(); err != nil {
			b.Fatalf("Close: %v", err)
		}
		b.StartTimer()
	}
	// })

	b.Run("Put", func(b *testing.B) {
		filePath := filepath.Join(b.TempDir(), "oram-latency.db")

		store, err := enigmap.NewORAM[oramLatencyValue](
			filePath,
			omapSize,
			[]byte("0123456789abcdef0123456789abcdef"),
		)
		if err != nil {
			b.Fatalf("NewORAM: %v", err)
		}
		defer store.Close()

		b.ResetTimer()

		for index := range b.N {
			if err := store.Put(
				oramLatencyKey(index),
				oramLatencyData("data", index),
			); err != nil {
				b.Fatalf("Put(%d): %v", index, err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		filePath := filepath.Join(b.TempDir(), "oram-latency.db")

		store, err := enigmap.NewORAM[oramLatencyValue](
			filePath,
			omapSize,
			[]byte("0123456789abcdef0123456789abcdef"),
		)
		if err != nil {
			b.Fatalf("NewORAM: %v", err)
		}
		defer store.Close()

		for index := 0; index < 10; index++ {
			if err := store.Put(
				oramLatencyKey(index),
				oramLatencyData("data", index),
			); err != nil {
				b.Fatalf("setup Put(%d): %v", index, err)
			}
		}

		b.ResetTimer()

		for index := range b.N {
			keyIndex := index % 10

			_, found, err := store.Get(oramLatencyKey(keyIndex))
			if err != nil {
				b.Fatalf("Get(%d): %v", keyIndex, err)
			}
			if !found {
				b.Fatalf("Get(%d): key not found", keyIndex)
			}
		}
	})
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
