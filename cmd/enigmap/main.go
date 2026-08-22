package main

import (
	"flag"
	"fmt"
	"log"
)

type runConfig struct {
	access     string
	storage    string
	encryption string
	filePath   string
}

func main() {
	access := flag.String("access", "direct", "access method: direct or oram")
	storage := flag.String("storage", "memory", "physical storage: memory or file")
	encryption := flag.String("encryption", "none", "storage encryption: none or aes-gcm")
	filePath := flag.String("file", "enigmap-oram.db", "file storage path")
	flag.Parse()

	config := runConfig{
		access:     *access,
		storage:    *storage,
		encryption: *encryption,
		filePath:   *filePath,
	}
	if err := run(config); err != nil {
		log.Fatal(err)
	}
}

func run(config runConfig) error {
	// Supported configurations are direct/memory/none, oram/memory/aes-gcm,
	// and oram/file/aes-gcm. Other combinations are not wired into this CLI.
	switch {
	case config.access == "direct" &&
		config.storage == "memory" &&
		config.encryption == "none":
		return runDirectMemory()
	case config.access == "oram" &&
		config.storage == "memory" &&
		config.encryption == "aes-gcm":
		return runORAMMemoryEncrypted()
	case config.access == "oram" &&
		config.storage == "file" &&
		config.encryption == "aes-gcm":
		return runORAMFile(config.filePath)
	default:
		return fmt.Errorf(
			"unsupported configuration: access=%s storage=%s encryption=%s",
			config.access,
			config.storage,
			config.encryption,
		)
	}
}
