package main

import (
	"fmt"

	enigmap "github.com/adely-rya/enigmap-go"
)

func main() {
	store := enigmap.NewMemory[string]()

	fmt.Printf("initialized %T\n", store)
}
