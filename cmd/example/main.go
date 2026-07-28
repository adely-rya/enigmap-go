package main

import (
	"crypto/subtle"
	"fmt"
	"log"

	enigmap "github.com/adely-rya/enigmap-go"
	"github.com/adely-rya/enigmap-go/omap"
)

type Value [32]byte

func main() {
	selector := omap.ValueSelectorFunc[Value](
		func(current, candidate Value, condition int) Value {
			for i := range current {
				current[i] = byte(subtle.ConstantTimeSelect(
					condition,
					int(candidate[i]),
					int(current[i]),
				))
			}
			return current
		},
	)

	store, err := enigmap.NewMemory[Value](1024, selector)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("initialized %T\n", store)
}
