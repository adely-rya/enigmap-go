package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	backend := flag.String("backend", "memory", "backend to run: memory or oram")
	filePath := flag.String("file", "enigmap-oram.db", "ORAM storage file path")
	flag.Parse()

	var err error
	switch *backend {
	case "memory":
		err = runMemory()
	case "oram":
		err = runORAM(*filePath)
	default:
		err = fmt.Errorf("unknown backend %q: use memory or oram", *backend)
	}

	if err != nil {
		log.Fatal(err)
	}
}
