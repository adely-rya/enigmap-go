package eoram

// Stash holds entries that have been read from ORAM paths but have not yet
// been written back. T will be the ENIGMAP-specific stashed block type.
type Stash[T any] struct {
	blocks []T
}
