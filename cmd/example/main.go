package main

import (
	"bytes"
	"fmt"
	"log"

	enigmap "github.com/adely-rya/enigmap-go"
	"github.com/adely-rya/enigmap-go/avl"
)

type Value [32]byte

const entryCount = 1000

func fixedKey(text string) []byte {
	key := make([]byte, avl.KeySize)
	copy(key, text)
	return key
}

func fixedValue(text string) Value {
	var value Value
	copy(value[:], text)
	return value
}

func main() {
	store, err := enigmap.NewMemory[Value](1024)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < entryCount; i++ {
		key := fixedKey(fmt.Sprintf("key-%04d", i))
		value := fixedValue(fmt.Sprintf("data-%04d", i))
		if err := store.Put(key, value); err != nil {
			log.Fatalf("Put failed at index %d: %v", i, err)
		}
	}

	// 同じkeyへもう一度Putし、既存valueを上書きする。
	for i := 0; i < entryCount; i++ {
		key := fixedKey(fmt.Sprintf("key-%04d", i))
		updatedValue := fixedValue(fmt.Sprintf("updated-data-%04d", i))
		if err := store.Put(key, updatedValue); err != nil {
			log.Fatalf("overwrite Put failed at index %d: %v", i, err)
		}
	}

	for i := 0; i < entryCount; i++ {
		key := fixedKey(fmt.Sprintf("key-%04d", i))
		want := fixedValue(fmt.Sprintf("updated-data-%04d", i))

		got, found, err := store.Get(key)
		if err != nil {
			log.Fatalf("Get failed at index %d: %v", i, err)
		}
		if !found {
			log.Fatalf("Get failed at index %d: inserted key was not found", i)
		}
		if !bytes.Equal(got[:], want[:]) {
			log.Fatalf(
				"Get returned unexpected value at index %d: got %q, want %q",
				i,
				got,
				want,
			)
		}
	}

	fmt.Printf("Put/overwrite/Get succeeded for all %d entries\n", entryCount)
}
