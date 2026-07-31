package main

import (
	"bytes"
	"crypto/subtle"
	"fmt"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/enigmap-go/omap"
	"github.com/adely-rya/oram/core"
)

const (
	entryCount = 1000
	maxNodes   = 1024
)

type exampleValue [eoram.ValueSize]byte

func (value exampleValue) Select(candidate exampleValue, condition int) exampleValue {
	var selected exampleValue
	for index := range selected {
		selected[index] = byte(subtle.ConstantTimeSelect(
			condition&1,
			int(candidate[index]),
			int(value[index]),
		))
	}
	return selected
}

func (value exampleValue) Serialize() ([]byte, error) {
	encoded := make([]byte, len(value))
	copy(encoded, value[:])
	return encoded, nil
}

func (exampleValue) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != len(exampleValue{}) {
		return nil, fmt.Errorf("example value size = %d, want %d", len(encoded), len(exampleValue{}))
	}
	var value exampleValue
	copy(value[:], encoded)
	return value, nil
}

func runExample(store omap.Store[exampleValue]) error {
	for index := 0; index < entryCount; index++ {
		key := fixedKey(fmt.Sprintf("key-%04d", index))
		value := fixedValue(fmt.Sprintf("data-%04d", index))
		if err := store.Put(key, value); err != nil {
			return fmt.Errorf("put index %d: %w", index, err)
		}
	}

	for index := 0; index < entryCount; index++ {
		key := fixedKey(fmt.Sprintf("key-%04d", index))
		value := fixedValue(fmt.Sprintf("updated-data-%04d", index))
		if err := store.Put(key, value); err != nil {
			return fmt.Errorf("overwrite index %d: %w", index, err)
		}
	}

	for index := 0; index < entryCount; index++ {
		key := fixedKey(fmt.Sprintf("key-%04d", index))
		want := fixedValue(fmt.Sprintf("updated-data-%04d", index))

		got, found, err := store.Get(key)
		if err != nil {
			return fmt.Errorf("get index %d: %w", index, err)
		}
		if !found {
			return fmt.Errorf("get index %d: key not found", index)
		}
		if !bytes.Equal(got[:], want[:]) {
			return fmt.Errorf("get index %d: got %q, want %q", index, got, want)
		}
	}

	return nil
}

func fixedKey(text string) []byte {
	key := make([]byte, avl.KeySize)
	copy(key, text)
	return key
}

func fixedValue(text string) exampleValue {
	var value exampleValue
	copy(value[:], text)
	return value
}
