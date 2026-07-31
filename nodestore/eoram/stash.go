package eoram

// StashedBlock wraps one ORAM block with ENIGMAP-specific stash state.
type StashedBlock[V any] struct {
	Block  Block[V]
	Sticky int
}

// Stash holds blocks that have been read from ORAM paths but have not yet been
// written back.
type Stash[V any] struct {
	blocks []StashedBlock[V]
}

func (block Block[V]) MakeStashedBlock(sticky int) StashedBlock[V] {
	return StashedBlock[V]{
		Block:  block,
		Sticky: sticky & 1,
	}
}
