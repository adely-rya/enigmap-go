package eoram

import "github.com/adely-rya/oram/core"

// StashedBlock wraps one ORAM block with ENIGMAP-specific stash state.
type StashedBlock[T core.Selectable[T]] struct {
	Block  Block[T]
	Sticky int
}

// Stash holds blocks that have been read from ORAM paths but have not yet been
// written back.
type Stash[T core.Selectable[T]] struct {
	blocks []StashedBlock[T]
}

func (block Block[T]) MakeStashedBlock(sticky int) StashedBlock[T] {
	return StashedBlock[T]{
		Block:  block,
		Sticky: sticky & 1,
	}
}
