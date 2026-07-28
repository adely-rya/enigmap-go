package main

import (
	"fmt"
	"log"

	enigmap "github.com/adely-rya/enigmap-go"
)

func main() {
	store, err := enigmap.NewMemory[string](1024)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("initialized %T\n", store)
}
