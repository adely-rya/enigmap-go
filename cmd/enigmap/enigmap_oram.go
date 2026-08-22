package main

import (
	"crypto/rand"
	"fmt"

	enigmap "github.com/adely-rya/enigmap-go"
)

func runORAMFile(filePath string) error {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate ORAM encryption key: %w", err)
	}

	store, err := enigmap.NewORAM[exampleValue](filePath, maxNodes, key)
	if err != nil {
		return fmt.Errorf("create ORAM map: %w", err)
	}
	defer store.Close()

	if err := runExample(store); err != nil {
		return fmt.Errorf("run ORAM map: %w", err)
	}

	fmt.Printf("oram/file/aes-gcm: Put/overwrite/Get succeeded for all %d entries\n", entryCount)
	return nil
}
