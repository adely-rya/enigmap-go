// //go:build latency

package enigmap_test

// import (
// 	"fmt"
// 	"os"
// 	"runtime"
// 	"testing"
// 	"time"

// 	enigmap "github.com/adely-rya/enigmap-go"
// )

// var oramLatencyEntryCount = time.Duration(omapSize)

// func TestORAMCacheProfile(t *testing.T) {
// 	for _, capacity := range []int{0, 8, 30, 85} {
// 		t.Run(fmt.Sprintf("capacity-%d", capacity), func(t *testing.T) {
// 			store, err := enigmap.NewORAMWithCacheCapacity[oramLatencyValue](
// 				t.TempDir()+"/oram.db",
// 				oramLatencyEntryCount,
// 				[]byte("0123456789abcdef0123456789abcdef"),
// 				capacity,
// 			)
// 			if err != nil {
// 				t.Fatalf("NewORAMWithCacheCapacity: %v", err)
// 			}

// 			started := time.Now()
// 			for index := 0; index < oramLatencyEntryCount; index++ {
// 				if err := store.Put(oramLatencyKey(index), oramLatencyData("data", index)); err != nil {
// 					t.Fatalf("Put(%d): %v", index, err)
// 				}
// 			}
// 			elapsed := time.Since(started)
// 			stats := store.Stats()
// 			if err := store.Close(); err != nil {
// 				t.Fatalf("Close: %v", err)
// 			}

// 			client := stats.Client
// 			cache := stats.Cache
// 			measured := client.ReadPathTime + client.AddPathTime +
// 				client.PrepareAccessTime + client.UpdateBlockTime +
// 				client.EvictToPathTime + client.WritePathTime + client.UnpinTime
// 			fmt.Fprintf(os.Stdout,
// 				"\ncapacity=%d Put=%s (%s/op) accesses=%d max-stash=%d hits=%d misses=%d hit-rate=%.1f%% reads=%d writes=%d dirty-evictions=%d\n",
// 				capacity, elapsed, elapsed/oramLatencyEntryCount, client.Accesses,
// 				client.MaxStashLength,
// 				cache.Hits, cache.Misses, percent(cache.Hits, cache.Hits+cache.Misses),
// 				cache.BackingReads, cache.BackingWrites, cache.DirtyEvictions,
// 			)
// 			fmt.Fprintf(os.Stdout,
// 				"read-path=%s add-path=%s prepare=%s update=%s evict-to-path=%s write-path=%s unpin=%s other=%s backing-read=%s backing-write=%s\n",
// 				client.ReadPathTime, client.AddPathTime, client.PrepareAccessTime,
// 				client.UpdateBlockTime, client.EvictToPathTime, client.WritePathTime,
// 				client.UnpinTime, elapsed-measured,
// 				cache.BackingReadTime, cache.BackingWriteTime,
// 			)
// 		})
// 	}
// }

// func TestORAMStashProfile(t *testing.T) {
// 	for _, stashSize := range []int{64, 96, 128, 179} {
// 		t.Run(fmt.Sprintf("stash-%d", stashSize), func(t *testing.T) {
// 			store, err := enigmap.NewORAMForProfile[oramLatencyValue](
// 				t.TempDir()+"/oram.db", oramLatencyEntryCount,
// 				[]byte("0123456789abcdef0123456789abcdef"), 85, stashSize,
// 			)
// 			if err != nil {
// 				t.Skipf("configuration rejected: %v", err)
// 			}
// 			var before, after runtime.MemStats
// 			runtime.GC()
// 			runtime.ReadMemStats(&before)
// 			started := time.Now()
// 			for index := 0; index < oramLatencyEntryCount; index++ {
// 				if err := store.Put(oramLatencyKey(index), oramLatencyData("data", index)); err != nil {
// 					_ = store.Close()
// 					t.Skipf("Put failed at %d: %v", index, err)
// 				}
// 			}
// 			elapsed := time.Since(started)
// 			runtime.ReadMemStats(&after)
// 			stats := store.Stats().Client
// 			if err := store.Close(); err != nil {
// 				t.Fatalf("Close: %v", err)
// 			}
// 			fmt.Fprintf(os.Stdout,
// 				"\nstash=%d Put=%s (%s/op) add-path=%s evict-to-path=%s alloc=%.1fMiB mallocs=%d\n",
// 				stashSize, elapsed, elapsed/oramLatencyEntryCount,
// 				stats.AddPathTime, stats.EvictToPathTime,
// 				float64(after.TotalAlloc-before.TotalAlloc)/(1024*1024),
// 				after.Mallocs-before.Mallocs,
// 			)
// 		})
// 	}
// }

// func percent(part, total uint64) float64 {
// 	if total == 0 {
// 		return 0
// 	}
// 	return 100 * float64(part) / float64(total)
// }
