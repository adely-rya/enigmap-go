package eoram

// StashedBlock wraps one ORAM block with ENIGMAP-specific stash state.
type StashedBlock[T any] struct {
	Block  T
	Sticky int
}

// Stash holds blocks that have been read from ORAM paths but have not yet been
// written back.
type Stash[T any] struct {
	blocks []StashedBlock[T]
}
