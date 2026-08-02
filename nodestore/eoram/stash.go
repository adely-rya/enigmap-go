package eoram

import (
	"crypto/subtle"
	"unsafe"

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
	blocks        []StashedBlock[T]
	capacity      int
	fixedCapacity bool
}

func (block Block[T]) MakeStashedBlock(sticky int) StashedBlock[T] {
	return StashedBlock[T]{
		Block:  block,
		Sticky: sticky & 1,
	}
}

func (stash *Stash[T]) Put(block Block[T]) int {
	exists := 0

	//同じブロックがスタッシュにあるかの判定
	for index := 0; index < len(stash.blocks); index++ {
		current := stash.blocks[index]
		exists |= block.Valid &
			current.Block.Valid &
			utils.CtEqUint64(
				uint64(current.Block.BlockAddress),
				uint64(block.BlockAddress),
			)
	}

	written := utils.CtNot(block.Valid)
	inserted := 0
	for index := 0; index < len(stash.blocks); index++ {
		current := stash.blocks[index]

		match := current.Block.Valid & utils.CtEqUint64(
			uint64(current.Block.BlockAddress),
			uint64(block.BlockAddress),
		) //同じアドレスでエントリーがリアルか判定
		update := exists & match
		insert := utils.CtNot(exists) & block.Valid &
			utils.CtNot(current.Block.Valid) & utils.CtNot(inserted)
		current.Block = current.Block.Select(block, update|insert)
		current.Sticky = subtle.ConstantTimeSelect(insert, 0, current.Sticky)
		stash.blocks[index] = current
		inserted |= insert
		written |= update | insert
	}

	if !stash.fixedCapacity && exists == 0 && block.Valid != 0 {
		if stash.capacity > 0 && len(stash.blocks) >= stash.capacity {
			return 0
		}
		stash.blocks = append(stash.blocks, block.MakeStashedBlock(0))
		written = 1
	}
	return written
}

// PutPinned adds a block to the stash and pins it against eviction.
func (stash *Stash[T]) PutPinned(block Block[T]) int {
	if stash.Put(block) == 0 {
		return 0
	}

	found := 0
	for index := 0; index < len(stash.blocks); index++ {
		entry := stash.blocks[index]
		match := entry.Block.Valid &
			utils.CtEqUint64(
				uint64(entry.Block.BlockAddress),
				uint64(block.BlockAddress),
			) &
			utils.CtNot(found)

		entry.Sticky = subtle.ConstantTimeSelect(match, 1, entry.Sticky)
		stash.blocks[index] = entry
		found |= match
	}

	return found
}

func (stash *Stash[T]) Get(blockAddress BlockAddress) (T, int) {
	var out T
	found := 0

	for index := 0; index < len(stash.blocks); index++ {
		block := stash.blocks[index].Block

		match := block.Valid &
			utils.CtEqUint64(
				uint64(block.BlockAddress),
				uint64(blockAddress),
			) &
			utils.CtNot(found)

		out = core.Select(out, block.Data, match)
		found |= match
	}

	return out, found
}

// PrepareAccess returns the requested block while assigning it to newLeaf
// and pinning it in the stash until FinishAccess completes.
func (stash *Stash[T]) PrepareAccess(
	blockAddress BlockAddress,
	newLeaf Leaf,
) (T, int) {
	var value T
	found := 0
	selected := 0

	for index := 0; index < len(stash.blocks); index++ {
		entry := stash.blocks[index]
		match := entry.Block.Valid &
			utils.CtEqUint64(
				uint64(entry.Block.BlockAddress),
				uint64(blockAddress),
			) &
			utils.CtNot(selected)

		value = core.Select(value, entry.Block.Data, match)
		entry.Block.Leaf = Leaf(utils.CtSelect64(
			match,
			uint64(newLeaf),
			uint64(entry.Block.Leaf),
		))
		entry.Sticky = subtle.ConstantTimeSelect(match, 1, entry.Sticky)
		stash.blocks[index] = entry
		selected |= match
		found |= match
	}

	return value, found
}

// UpdateAccessedBlock replaces the payload of one valid stash entry.
func (stash *Stash[T]) UpdateAccessedBlock(
	blockAddress BlockAddress,
	data T,
) int {
	updated := 0

	for index := 0; index < len(stash.blocks); index++ {
		entry := stash.blocks[index]
		match := entry.Block.Valid &
			utils.CtEqUint64(
				uint64(entry.Block.BlockAddress),
				uint64(blockAddress),
			) &
			utils.CtNot(updated)

		entry.Block.Data = core.Select(entry.Block.Data, data, match)
		stash.blocks[index] = entry
		updated |= match
	}

	return updated
}

// Unpin makes one valid stash entry eligible for a later eviction.
func (stash *Stash[T]) Unpin(blockAddress BlockAddress) int {
	found := 0

	for index := 0; index < len(stash.blocks); index++ {
		entry := stash.blocks[index]
		match := entry.Block.Valid &
			utils.CtEqUint64(
				uint64(entry.Block.BlockAddress),
				uint64(blockAddress),
			) &
			utils.CtNot(found)

		entry.Sticky = subtle.ConstantTimeSelect(match, 0, entry.Sticky)
		stash.blocks[index] = entry
		found |= match
	}

	return found
}

// UnpinAll makes every stash entry eligible for later eviction.
func (stash *Stash[T]) UnpinAll() {
	for index := 0; index < len(stash.blocks); index++ {
		stash.blocks[index].Sticky = 0
	}
}

// FreeMaybe clears one valid block when condition is 1. The stash is fully
// scanned regardless of condition and block location.
func (stash *Stash[T]) FreeMaybe(
	blockAddress BlockAddress,
	condition int,
) int {
	condition &= 1
	found := 0
	var empty Block[T]

	for index := 0; index < len(stash.blocks); index++ {
		entry := stash.blocks[index]
		match := entry.Block.Valid &
			utils.CtEqUint64(
				uint64(entry.Block.BlockAddress),
				uint64(blockAddress),
			) &
			utils.CtNot(found)
		free := match & condition

		entry.Block = entry.Block.Select(empty, free)
		entry.Sticky = subtle.ConstantTimeSelect(free, 0, entry.Sticky)
		stash.blocks[index] = entry
		found |= match
	}

	return found
}

func (stash *Stash[T]) AddPath(path []Bucket[T], bucketSize int) int {
	for depth := 0; depth < len(path); depth++ {
		if len(path[depth]) != bucketSize {
			return 0
		}
	}
	if fixedStash, ok := any(stash).(*Stash[NodeBlock[FixedValue]]); ok && stash.fixedCapacity {
		return addFixedValuePath(
			fixedStash,
			any(path).([]Bucket[NodeBlock[FixedValue]]),
			bucketSize,
		)
	}
	if !stash.fixedCapacity {
		required := len(path) * bucketSize
		if stash.capacity > 0 && len(stash.blocks)+required > stash.capacity {
			return 0
		}
		for depth := 0; depth < len(path); depth++ {
			for slot := 0; slot < bucketSize; slot++ {
				stash.blocks = append(
					stash.blocks,
					path[depth][slot].MakeStashedBlock(0),
				)
			}
		}
		return 1
	}
	added := 1
	for depth := 0; depth < len(path); depth++ {
		for slot := 0; slot < bucketSize; slot++ {
			added &= stash.Put(path[depth][slot])
		}
	}
	return added
}

// addFixedValuePath preserves the fixed two-pass stash trace while replacing
// nested generic field selection with raw word selection of the pointer-free
// concrete block.
func addFixedValuePath(
	stash *Stash[NodeBlock[FixedValue]],
	path []Bucket[NodeBlock[FixedValue]],
	bucketSize int,
) int {
	const blockWordCount = int(unsafe.Sizeof(Block[NodeBlock[FixedValue]]{})) / 8
	added := 1
	for depth := 0; depth < len(path); depth++ {
		for slot := 0; slot < bucketSize; slot++ {
			block := path[depth][slot]
			exists := 0
			for index := 0; index < len(stash.blocks); index++ {
				current := stash.blocks[index]
				exists |= block.Valid & current.Block.Valid & utils.CtEqUint64(
					uint64(current.Block.BlockAddress),
					uint64(block.BlockAddress),
				)
			}
			written := utils.CtNot(block.Valid)
			inserted := 0
			for index := 0; index < len(stash.blocks); index++ {
				current := &stash.blocks[index]
				match := current.Block.Valid & utils.CtEqUint64(
					uint64(current.Block.BlockAddress),
					uint64(block.BlockAddress),
				)
				update := exists & match
				insert := utils.CtNot(exists) & block.Valid &
					utils.CtNot(current.Block.Valid) & utils.CtNot(inserted)
				take := update | insert
				currentWords := (*[blockWordCount]uint64)(unsafe.Pointer(&current.Block))
				candidateWords := (*[blockWordCount]uint64)(unsafe.Pointer(&block))
				for word := 0; word < blockWordCount; word++ {
					currentWords[word] = selectUint64(
						currentWords[word],
						candidateWords[word],
						take,
					)
				}
				current.Sticky = int(selectUint64(
					uint64(current.Sticky),
					0,
					insert,
				))
				inserted |= insert
				written |= take
			}
			added &= written
		}
	}
	return added
}

func (stash *Stash[T]) validBlockCount() int {
	count := 0
	for index := 0; index < len(stash.blocks); index++ {
		count += stash.blocks[index].Block.Valid & 1
	}
	return count
}

// EvictToPath greedily fills a root-to-leaf path from the deepest bucket.
// Sticky entries remain in the stash even when their position is on the path.
func (stash *Stash[T]) EvictToPath(
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[T] {
	if fixedStash, ok := any(stash).(*Stash[NodeBlock[FixedValue]]); ok {
		fixedPath := evictFixedValueToPath(
			fixedStash,
			leaf,
			pathLength,
			bucketSize,
		)
		return any(fixedPath).([]Bucket[T])
	}

	path := make([]Bucket[T], pathLength)
	blocks := make([]Block[T], pathLength*bucketSize)
	used := make([]int, bucketSize)
	for depth := 0; depth < pathLength; depth++ {
		start := depth * bucketSize
		path[depth] = blocks[start : start+bucketSize]
	}

	for depth := pathLength - 1; depth >= 0; depth-- {
		shift := uint(pathLength - 1 - depth)
		isRoot := utils.CtEqUint64(uint64(depth), 0)

		for slot := 0; slot < bucketSize; slot++ {
			used[slot] = 0
		} //空のroot→leaf pathを作る

		for index := 0; index < len(stash.blocks); index++ {
			current := stash.blocks[index]

			blockLeaf := uint64(current.Block.Leaf)
			prefixMatch := utils.CtEqUint64(
				blockLeaf>>shift,
				uint64(leaf)>>shift,
			)
			onPath := isRoot | (utils.CtNot(isRoot) & prefixMatch)

			eligible := current.Block.Valid & utils.CtNot(current.Sticky) & onPath
			placed := 0

			for slot := 0; slot < bucketSize; slot++ {
				free := utils.CtNot(used[slot])
				take := eligible & utils.CtNot(placed) & free

				path[depth][slot] = path[depth][slot].Select(
					current.Block,
					take,
				)

				used[slot] |= take
				placed |= take
			}

			current.Block.Valid = subtle.ConstantTimeSelect(
				placed,
				0,
				current.Block.Valid,
			)
			stash.blocks[index] = current
		}
	}
	stash.compact()

	return path
}

// evictFixedValueToPath is the concrete, dictionary-free strong-oblivious
// eviction path for the configured eight-byte key/value representation.
func evictFixedValueToPath(
	stash *Stash[NodeBlock[FixedValue]],
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[NodeBlock[FixedValue]] {
	const blockWordCount = int(unsafe.Sizeof(Block[NodeBlock[FixedValue]]{})) / 8
	path := make([]Bucket[NodeBlock[FixedValue]], pathLength)
	blocks := make([]Block[NodeBlock[FixedValue]], pathLength*bucketSize)
	used := make([]int, bucketSize)
	for depth := 0; depth < pathLength; depth++ {
		start := depth * bucketSize
		path[depth] = blocks[start : start+bucketSize]
	}
	for depth := pathLength - 1; depth >= 0; depth-- {
		shift := uint(pathLength - 1 - depth)
		isRoot := utils.CtEqUint64(uint64(depth), 0)
		for slot := 0; slot < bucketSize; slot++ {
			used[slot] = 0
		}
		for index := 0; index < len(stash.blocks); index++ {
			current := stash.blocks[index]
			prefixMatch := utils.CtEqUint64(
				uint64(current.Block.Leaf)>>shift,
				uint64(leaf)>>shift,
			)
			onPath := isRoot | (utils.CtNot(isRoot) & prefixMatch)
			eligible := current.Block.Valid & utils.CtNot(current.Sticky) & onPath
			placed := 0
			for slot := 0; slot < bucketSize; slot++ {
				free := utils.CtNot(used[slot])
				take := eligible & utils.CtNot(placed) & free
				currentWords := (*[blockWordCount]uint64)(unsafe.Pointer(&path[depth][slot]))
				candidateWords := (*[blockWordCount]uint64)(unsafe.Pointer(&current.Block))
				for word := 0; word < blockWordCount; word++ {
					currentWords[word] = selectUint64(
						currentWords[word],
						candidateWords[word],
						take,
					)
				}
				used[slot] |= take
				placed |= take
			}
			current.Block.Valid = int(selectUint64(
				uint64(current.Block.Valid),
				0,
				placed,
			))
			stash.blocks[index] = current
		}
	}
	stash.compact()
	return path
}

// EvictToPathNonOblivious is an experiment-only counterpart matching the
// public C++ implementation's ObliviousCPUTrace=false branch. Its loop count
// and branches depend on stash contents, so it must not be used when strong
// obliviousness is required.
func (stash *Stash[T]) EvictToPathNonOblivious(
	leaf Leaf,
	pathLength int,
	bucketSize int,
) ([]Bucket[T], uint64) {
	path := make([]Bucket[T], pathLength)
	blocks := make([]Block[T], pathLength*bucketSize)
	for depth := 0; depth < pathLength; depth++ {
		start := depth * bucketSize
		path[depth] = blocks[start : start+bucketSize]
	}
	var stashVisits uint64

	for depth := pathLength - 1; depth >= 0; depth-- {
		shift := uint(pathLength - 1 - depth)
		slot := 0
		for index := 0; index < len(stash.blocks); index++ {
			if slot == bucketSize {
				break
			}
			stashVisits++
			entry := &stash.blocks[index]
			if entry.Block.Valid == 0 || entry.Sticky != 0 {
				continue
			}
			if depth != 0 && uint64(entry.Block.Leaf)>>shift != uint64(leaf)>>shift {
				continue
			}
			path[depth][slot] = entry.Block
			entry.Block.Valid = 0
			slot++
		}
	}
	stash.compact()
	return path, stashVisits
}

func (stash *Stash[T]) compact() {
	if stash.fixedCapacity {
		return
	}
	write := 0
	for read := 0; read < len(stash.blocks); read++ {
		if stash.blocks[read].Block.Valid == 0 {
			continue
		}
		stash.blocks[write] = stash.blocks[read]
		write++
	}
	var zero StashedBlock[T]
	for index := write; index < len(stash.blocks); index++ {
		stash.blocks[index] = zero
	}
	stash.blocks = stash.blocks[:write]
}
