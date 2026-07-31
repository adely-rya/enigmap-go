package eoram

import (
	"crypto/subtle"

	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/utils"
)

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

func (stash *Stash[T]) Put(block Block[T]) int {
	exists := 0

	for index := 0; index < len(stash.blocks); index++ {
		current := stash.blocks[index]
		exists |= block.Valid &
			current.Block.Valid &
			utils.CtEqUint64(
				uint64(current.Block.Address),
				uint64(block.Address),
			)
	}

	inserted := 0
	written := exists
	for index := 0; index < len(stash.blocks); index++ {
		current := stash.blocks[index]

		match := current.Block.Valid & utils.CtEqUint64(
			uint64(current.Block.Address),
			uint64(block.Address),
		)
		update := exists & match
		free := utils.CtNot(current.Block.Valid)
		take := block.Valid &
			utils.CtNot(exists) &
			utils.CtNot(inserted) &
			free

		write := update | take
		current.Block = current.Block.Select(block, write)
		current.Sticky = subtle.ConstantTimeSelect(
			take,
			0,
			current.Sticky,
		)
		stash.blocks[index] = current

		inserted |= take
		written |= write
	}

	return utils.CtNot(block.Valid) | written
}

func (stash *Stash[T]) AddPath(path []Bucket[T], bucketSize int) int {
	ok := 1

	for depth := 0; depth < len(path); depth++ {
		if len(path[depth]) != bucketSize {
			return 0
		}

		for slot := 0; slot < bucketSize; slot++ {
			ok = stash.Put(path[depth][slot]) & ok
		}
	}

	return ok
}
