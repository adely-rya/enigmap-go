package main

import (
	"fmt"
	"log"

	enigmap "github.com/adely-rya/enigmap-go"
)

type Value [32]byte

func main() {
	store, err := enigmap.NewMemory[Value](1024)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("initialized %T\n", store)
}
