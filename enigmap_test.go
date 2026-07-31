package enigmap

import (
	"bytes"
	"crypto/subtle"
	"fmt"
	"os"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/oram/core"
)

type selectableValue [eoram.ValueSize]byte

var testEncryptionKey = []byte("0123456789abcdef0123456789abcdef")

func (value selectableValue) Select(
	candidate selectableValue,
	condition int,
) selectableValue {
	var selected selectableValue
	for index := range selected {
		selected[index] = byte(subtle.ConstantTimeSelect(
			condition&1,
			int(candidate[index]),
			int(value[index]),
		))
	}
	return selected
}

func TestORAMStorePutGetOnFile(t *testing.T) {
	store, err := NewORAM[selectableValue](t.TempDir()+"/oram.db", 16, testEncryptionKey)
	if err != nil {
		t.Fatalf("NewORAM: %v", err)
	}
	defer store.Close()

	for index := 0; index < 10; index++ {
		key := make([]byte, avl.KeySize)
		copy(key, fmt.Sprintf("key-%02d", index))
		var value selectableValue
		value[0] = byte(index + 1)
		if err := store.Put(key, value); err != nil {
			t.Fatalf("Put(%d): %v", index, err)
		}
	}

	for index := 0; index < 10; index++ {
		key := make([]byte, avl.KeySize)
		copy(key, fmt.Sprintf("key-%02d", index))
		value, found, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%d): %v", index, err)
		}
		if !found || value[0] != byte(index+1) {
			t.Fatalf("Get(%d) = (%d, %v), want (%d, true)", index, value[0], found, index+1)
		}
	}
}

func (value selectableValue) Serialize() ([]byte, error) {
	encoded := make([]byte, len(value))
	copy(encoded, value[:])
	return encoded, nil
}

func (selectableValue) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != eoram.ValueSize {
		return nil, fmt.Errorf("value size = %d, want %d", len(encoded), eoram.ValueSize)
	}
	var value selectableValue
	copy(value[:], encoded)
	return value, nil
}

func TestNewORAMCreatesStore(t *testing.T) {
	got, err := NewORAM[selectableValue](t.TempDir()+"/oram.db", 3, testEncryptionKey)
	if err != nil {
		t.Fatalf("NewORAM: %v", err)
	}
	if got == nil {
		t.Fatal("NewORAM returned nil store")
	}
	if err := got.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewORAMRejectsInvalidEncryptionKey(t *testing.T) {
	store, err := NewORAM[selectableValue](t.TempDir()+"/oram.db", 3, []byte("short"))
	if err == nil {
		if store != nil {
			_ = store.Close()
		}
		t.Fatal("NewORAM accepted an invalid AES key")
	}
}

func TestORAMFileDoesNotContainPlaintext(t *testing.T) {
	filePath := t.TempDir() + "/oram.db"
	store, err := NewORAM[selectableValue](filePath, 3, testEncryptionKey)
	if err != nil {
		t.Fatalf("NewORAM: %v", err)
	}

	keyText := []byte("distinct-key-material")
	key := make([]byte, avl.KeySize)
	copy(key, keyText)
	valueText := []byte("distinct-value-material")
	var value selectableValue
	copy(value[:], valueText)
	if err := store.Put(key, value); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if bytes.Contains(contents, keyText) || bytes.Contains(contents, valueText) {
		t.Fatal("ORAM file contains plaintext key or value")
	}
}

func TestORAMDetectsTamperedCiphertext(t *testing.T) {
	filePath := t.TempDir() + "/oram.db"
	store, err := NewORAM[selectableValue](filePath, 3, testEncryptionKey)
	if err != nil {
		t.Fatalf("NewORAM: %v", err)
	}
	defer store.Close()

	key := make([]byte, avl.KeySize)
	copy(key, "tamper-test-key")
	var value selectableValue
	copy(value[:], "tamper-test-value")
	if err := store.Put(key, value); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.largeBucketStore.FlushAndClear(); err != nil {
		t.Fatalf("FlushAndClear: %v", err)
	}

	raw, err := os.OpenFile(filePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	var firstByte [1]byte
	if _, err := raw.ReadAt(firstByte[:], 0); err != nil {
		_ = raw.Close()
		t.Fatalf("ReadAt: %v", err)
	}
	firstByte[0] ^= 0xff
	if _, err := raw.WriteAt(firstByte[:], 0); err != nil {
		_ = raw.Close()
		t.Fatalf("WriteAt: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close tamper handle: %v", err)
	}

	if _, _, err := store.Get(key); err == nil {
		t.Fatal("Get accepted tampered ciphertext")
	}
}
