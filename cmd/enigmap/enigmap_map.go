package main

import (
	"fmt"

	enigmap "github.com/adely-rya/enigmap-go"
)

func runDirectMemory() error {
	store, err := enigmap.NewMemory[exampleValue](maxNodes)
	if err != nil {
		return fmt.Errorf("create memory map: %w", err)
	}

	if err := runExample(store); err != nil {
		return fmt.Errorf("run memory map: %w", err)
	}

	fmt.Printf("direct/memory/none: Put/overwrite/Get succeeded for all %d entries\n", entryCount)
	return nil
}
