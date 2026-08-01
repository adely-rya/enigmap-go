//go:build latency

package enigmap_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	enigmap "github.com/adely-rya/enigmap-go"
)

func TestORAMHotLoopExperiment(t *testing.T) {
	for _, entries := range []int{1000, 3000} {
		for _, cacheCapacity := range []int{30, 8192} {
			name := fmt.Sprintf("entries-%d/cache-%d", entries, cacheCapacity)
			t.Run(name, func(t *testing.T) {
				store, err := enigmap.NewORAMWithCacheCapacity[oramLatencyValue](
					t.TempDir()+"/oram.db",
					uint64(entries),
					[]byte("0123456789abcdef0123456789abcdef"),
					cacheCapacity,
				)
				if err != nil {
					t.Fatalf("NewORAMWithCacheCapacity: %v", err)
				}

				started := time.Now()
				for index := 0; index < entries; index++ {
					if err := store.Put(
						oramLatencyKey(index),
						oramLatencyData("data", index),
					); err != nil {
						t.Fatalf("Put(%d): %v", index, err)
					}
				}
				elapsed := time.Since(started)
				stats := store.Stats()
				if err := store.Close(); err != nil {
					t.Fatalf("Close: %v", err)
				}

				client := stats.Client
				cache := stats.Cache
				fmt.Fprintf(os.Stdout,
					"\nHOTLOOP entries=%d cache=%d put=%s op=%s accesses=%d max-stash=%d\n",
					entries, cacheCapacity, elapsed, elapsed/time.Duration(entries),
					client.Accesses, client.MaxStashLength,
				)
				fmt.Fprintf(os.Stdout,
					"times read=%s add=%s prepare=%s update=%s evict=%s write=%s unpin=%s\n",
					client.ReadPathTime, client.AddPathTime, client.PrepareAccessTime,
					client.UpdateBlockTime, client.EvictToPathTime,
					client.WritePathTime, client.UnpinTime,
				)
				fmt.Fprintf(os.Stdout,
					"loops path-buckets=%d add-blocks=%d prepare-scans=%d update-scans=%d evict-scans=%d evict-selects=%d unpin-scans=%d\n",
					client.PathBucketsRead, client.AddPathBlocks,
					client.PrepareStashScans, client.UpdateStashScans,
					client.EvictStashScans, client.EvictBlockSelects,
					client.UnpinStashScans,
				)
				fmt.Fprintf(os.Stdout,
					"cache hits=%d misses=%d hit-rate=%.1f%% reads=%d writes=%d dirty-evictions=%d read-time=%s write-time=%s\n",
					cache.Hits, cache.Misses, percent(cache.Hits, cache.Hits+cache.Misses),
					cache.BackingReads, cache.BackingWrites, cache.DirtyEvictions,
					cache.BackingReadTime, cache.BackingWriteTime,
				)
			})
		}
	}
}

func TestORAMEncryptionExperiment(t *testing.T) {
	const entries = 1000
	for _, encrypted := range []bool{true, false} {
		name := fmt.Sprintf("encrypted-%t", encrypted)
		t.Run(name, func(t *testing.T) {
			var store *enigmap.ORAMMap[oramLatencyValue]
			var err error
			if encrypted {
				store, err = enigmap.NewORAMWithCacheCapacity[oramLatencyValue](
					t.TempDir()+"/oram.db", entries,
					[]byte("0123456789abcdef0123456789abcdef"), 30,
				)
			} else {
				store, err = enigmap.NewORAMForProfileUnencrypted[oramLatencyValue](
					t.TempDir()+"/oram.db", entries, 30,
				)
			}
			if err != nil {
				t.Fatalf("create ORAM: %v", err)
			}

			started := time.Now()
			for index := 0; index < entries; index++ {
				if err := store.Put(oramLatencyKey(index), oramLatencyData("data", index)); err != nil {
					t.Fatalf("Put(%d): %v", index, err)
				}
			}
			elapsed := time.Since(started)
			stats := store.Stats()
			if err := store.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			fmt.Fprintf(os.Stdout,
				"\nENCRYPTION encrypted=%t put=%s op=%s evict=%s read-path=%s backing-read=%s backing-write=%s reads=%d writes=%d\n",
				encrypted, elapsed, elapsed/entries, stats.Client.EvictToPathTime,
				stats.Client.ReadPathTime, stats.Cache.BackingReadTime,
				stats.Cache.BackingWriteTime, stats.Cache.BackingReads,
				stats.Cache.BackingWrites,
			)
		})
	}
}

func TestORAMEvictionModeExperiment(t *testing.T) {
	for _, entries := range []int{1000, 3000} {
		for _, nonOblivious := range []bool{false, true} {
			name := fmt.Sprintf("entries-%d/non-oblivious-%t", entries, nonOblivious)
			t.Run(name, func(t *testing.T) {
				store, err := enigmap.NewORAMWithCacheCapacity[oramLatencyValue](
					t.TempDir()+"/oram.db", uint64(entries),
					[]byte("0123456789abcdef0123456789abcdef"), 8192,
				)
				if err != nil {
					t.Fatalf("create ORAM: %v", err)
				}
				if nonOblivious {
					store.EnableNonObliviousEvictionForProfile()
				}
				started := time.Now()
				for index := 0; index < entries; index++ {
					if err := store.Put(oramLatencyKey(index), oramLatencyData("data", index)); err != nil {
						t.Fatalf("Put(%d): %v", index, err)
					}
				}
				elapsed := time.Since(started)
				stats := store.Stats().Client
				if err := store.Close(); err != nil {
					t.Fatalf("Close: %v", err)
				}
				fmt.Fprintf(os.Stdout,
					"\nEVICTION entries=%d non-oblivious=%t put=%s op=%s evict=%s stash-visits=%d block-selects=%d max-stash=%d\n",
					entries, nonOblivious, elapsed, elapsed/time.Duration(entries),
					stats.EvictToPathTime, stats.EvictStashScans,
					stats.EvictBlockSelects, stats.MaxStashLength,
				)
			})
		}
	}
}
