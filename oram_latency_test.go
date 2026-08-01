//go:build latency

package enigmap_test

import (
	"crypto/subtle"
	"fmt"
	"os"
	"testing"
	"time"

	enigmap "github.com/adely-rya/enigmap-go"
	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/oram/core"
)

const (
	oramLatencyEntryCount = 5000
)

type oramLatencyValue [eoram.ValueSize]byte

func (value oramLatencyValue) Select(
	candidate oramLatencyValue,
	condition int,
) oramLatencyValue {
	var selected oramLatencyValue
	for index := range selected {
		selected[index] = byte(subtle.ConstantTimeSelect(
			condition&1,
			int(candidate[index]),
			int(value[index]),
		))
	}
	return selected
}

func (value oramLatencyValue) Serialize() ([]byte, error) {
	encoded := make([]byte, len(value))
	copy(encoded, value[:])
	return encoded, nil
}

func (oramLatencyValue) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != eoram.ValueSize {
		return nil, fmt.Errorf(
			"ORAM latency value size = %d, want %d",
			len(encoded),
			eoram.ValueSize,
		)
	}
	var value oramLatencyValue
	copy(value[:], encoded)
	return value, nil
}

func TestORAMFileStoreLatency(t *testing.T) {
	filePath := t.TempDir() + "/oram-latency.db"

	started := time.Now()
	store, err := enigmap.NewORAM[oramLatencyValue](
		filePath,
		oramLatencyEntryCount,
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("NewORAM: %v", err)
	}
	initialization := time.Since(started)
	defer store.Close()

	started = time.Now()
	for index := 0; index < oramLatencyEntryCount; index++ {
		if err := store.Put(
			oramLatencyKey(index),
			oramLatencyData("data", index),
		); err != nil {
			t.Fatalf("new Put(%d): %v", index, err)
		}
	}
	newPut := time.Since(started)

	started = time.Now()
	for index := 0; index < oramLatencyEntryCount; index++ {
		if err := store.Put(
			oramLatencyKey(index),
			oramLatencyData("updated", index),
		); err != nil {
			t.Fatalf("overwrite Put(%d): %v", index, err)
		}
	}
	overwrite := time.Since(started)

	started = time.Now()
	for index := 0; index < oramLatencyEntryCount; index++ {
		_, found, err := store.Get(oramLatencyKey(index))
		if err != nil {
			t.Fatalf("Get(%d): %v", index, err)
		}
		if !found {
			t.Fatalf("Get(%d): key not found", index)
		}
	}
	get := time.Since(started)

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "ORAM file latency (%d operations each)\n", oramLatencyEntryCount)
	fmt.Fprintf(os.Stdout, "initialization: %s\n", initialization)
	fmt.Fprintf(os.Stdout, "new Put:       %s total, %s/op\n", newPut, newPut/oramLatencyEntryCount)
	fmt.Fprintf(os.Stdout, "overwrite Put: %s total, %s/op\n", overwrite, overwrite/oramLatencyEntryCount)
	fmt.Fprintf(os.Stdout, "Get:           %s total, %s/op\n", get, get/oramLatencyEntryCount)
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
