//go:build latency

package enigmap_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	enigmap "github.com/adely-rya/enigmap-go"
	"github.com/adely-rya/enigmap-go/avl"
)

type latencyValue [32]byte

type latencyResult struct {
	entryCount        int
	newPutPerOp       time.Duration
	overwritePutPerOp time.Duration
	getPerOp          time.Duration
}

const (
	minimumLatencyRounds      = 5
	minimumMeasuredOperations = 10000
)

func latencyKey(index int) []byte {
	key := make([]byte, avl.KeySize)
	copy(key, fmt.Sprintf("key-%08d", index))
	return key
}

func latencyData(prefix string, index int) latencyValue {
	var value latencyValue
	copy(value[:], fmt.Sprintf("%s-%08d", prefix, index))
	return value
}

// TestMemoryStoreLatency は、メモリNodeStore上のAVL操作時間をデータ数ごとに計測する。
//
// ORAM・暗号化・ファイルIOはまだ含まれないため、最終的なENIGMAP性能ではなく、
// AVL実装の比較と性能回帰を確認するための基準値として使用する。
//
// 実行方法:
//
//	go test -tags latency -run TestMemoryStoreLatency -v
func TestMemoryStoreLatency(t *testing.T) {
	entryCounts := []int{10, 100, 1000, 10000, 100000, 1000000}
	results := make([]latencyResult, 0, len(entryCounts))

	for _, entryCount := range entryCounts {
		result := measureMemoryStoreLatency(t, entryCount)
		results = append(results, result)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(
		os.Stdout,
		"%10s | %14s | %14s | %14s\n",
		"entries",
		"new Put/op",
		"overwrite/op",
		"Get/op",
	)
	fmt.Fprintln(
		os.Stdout,
		"-----------+----------------+----------------+----------------",
	)
	for _, result := range results {
		fmt.Fprintf(
			os.Stdout,
			"%10d | %14s | %14s | %14s\n",
			result.entryCount,
			result.newPutPerOp,
			result.overwritePutPerOp,
			result.getPerOp,
		)
	}
}

// measureMemoryStoreLatency は、指定件数のTreeを毎round作り直して各操作を計測する。
// 時計の分解能による0秒を避けるため、小さいデータ数ほどround数を増やし、
// 各データ数で最低minimumMeasuredOperations回を集計してから1操作平均を返す。
func measureMemoryStoreLatency(t *testing.T, entryCount int) latencyResult {
	t.Helper()

	var newPutTotal time.Duration
	var overwritePutTotal time.Duration
	var getTotal time.Duration
	rounds := minimumMeasuredOperations / entryCount
	if rounds < minimumLatencyRounds {
		rounds = minimumLatencyRounds
	}

	for round := 0; round < rounds; round++ {
		store, err := enigmap.NewMemory[latencyValue](uint64(entryCount))
		if err != nil {
			t.Fatalf("NewMemory(%d): %v", entryCount, err)
		}

		started := time.Now()
		for index := 0; index < entryCount; index++ {
			key := latencyKey(index)
			value := latencyData("data", index)
			if err := store.Put(key, value); err != nil {
				t.Fatalf(
					"new Put: entries=%d round=%d index=%d: %v",
					entryCount,
					round,
					index,
					err,
				)
			}
		}
		newPutTotal += time.Since(started)

		started = time.Now()
		for index := 0; index < entryCount; index++ {
			key := latencyKey(index)
			value := latencyData("updated", index)
			if err := store.Put(key, value); err != nil {
				t.Fatalf(
					"overwrite Put: entries=%d round=%d index=%d: %v",
					entryCount,
					round,
					index,
					err,
				)
			}
		}
		overwritePutTotal += time.Since(started)

		started = time.Now()
		for index := 0; index < entryCount; index++ {
			key := latencyKey(index)
			want := latencyData("updated", index)
			got, found, err := store.Get(key)
			if err != nil {
				t.Fatalf(
					"Get: entries=%d round=%d index=%d: %v",
					entryCount,
					round,
					index,
					err,
				)
			}
			if !found {
				t.Fatalf(
					"Get: entries=%d round=%d index=%d: key not found",
					entryCount,
					round,
					index,
				)
			}
			if !bytes.Equal(got[:], want[:]) {
				t.Fatalf(
					"Get: entries=%d round=%d index=%d: got %q, want %q",
					entryCount,
					round,
					index,
					got,
					want,
				)
			}
		}
		getTotal += time.Since(started)
	}

	measuredOperations := time.Duration(entryCount * rounds)
	return latencyResult{
		entryCount:        entryCount,
		newPutPerOp:       newPutTotal / measuredOperations,
		overwritePutPerOp: overwritePutTotal / measuredOperations,
		getPerOp:          getTotal / measuredOperations,
	}
}
