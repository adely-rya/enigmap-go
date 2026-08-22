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

// addFixedValuePath concatenates the current fixed-capacity stash and the
// complete path into one public-size work buffer. The normal Client
// constructor reserves this tail capacity once, so the hot path only reslices
// the existing backing array. A heap growth remains as a compatibility
// fallback for manually constructed or unusually large profiling stashes.
func addFixedValuePath(
	stash *Stash[NodeBlock[FixedValue]],
	path []Bucket[NodeBlock[FixedValue]],
	bucketSize int,
) int {
	baseSize := stash.capacity
	if baseSize <= 0 {
		baseSize = len(stash.blocks)
	}
	if len(stash.blocks) != baseSize {
		return 0
	}

	pathBlockCount := len(path) * bucketSize
	workLength := baseSize + pathBlockCount
	if workLength < baseSize {
		return 0
	}
	if cap(stash.blocks) < workLength {
		work := make([]StashedBlock[NodeBlock[FixedValue]], workLength)
		copy(work[:baseSize], stash.blocks)
		stash.blocks = work
	} else {
		stash.blocks = stash.blocks[:workLength]
	}

	write := baseSize
	for depth := 0; depth < len(path); depth++ {
		for slot := 0; slot < bucketSize; slot++ {
			stash.blocks[write] = path[depth][slot].MakeStashedBlock(0)
			write++
		}
	}

	return 1
}

// fixedEvictionEntry mirrors EnigMap's TaggedT<StashedBlock_t>. specialDummy
// distinguishes the per-level counter sentinels (and ordinary invalid input
// blocks) from real entries and from padding dummies that must occupy path
// slots. tag is shallowness from the leaf: 0 is the leaf bucket and
// pathLength-1 is the root. pathLength itself is the overflow/stash tag.
type fixedEvictionEntry struct {
	stashed      StashedBlock[NodeBlock[FixedValue]]
	tag          uint64
	specialDummy int
}

const fixedEvictionEntrySize = int(unsafe.Sizeof(fixedEvictionEntry{}))
const fixedEvictionEntryWordCount = fixedEvictionEntrySize / 8

// Keep the fixed-sort scratch area below the compiler's large-stack-object
// threshold. With the current 128-byte entry this provides 768 entries, enough
// for the default 179-slot stash plus two complete 64-level, Z=4 path regions.
const fixedEvictionStackBudget = 96 << 10
const fixedEvictionStackEntryCapacity = fixedEvictionStackBudget / fixedEvictionEntrySize

var _ [-(fixedEvictionEntrySize % 8)]struct{}

const (
	fixedEvictionSortByTag = iota
	fixedEvictionSortCountersFirst
)

// ctLessUint64 returns 1 iff left < right without data-dependent control flow.
func ctLessUint64(left, right uint64) int {
	difference := left - right
	borrow := ((^left & right) | (^(left ^ right) & difference)) >> 63
	return int(borrow & 1)
}

func conditionalSwapFixedEvictionEntries(
	left *fixedEvictionEntry,
	right *fixedEvictionEntry,
	condition int,
) {
	mask := uint64(0) - uint64(condition&1)
	leftWords := (*[fixedEvictionEntryWordCount]uint64)(unsafe.Pointer(left))
	rightWords := (*[fixedEvictionEntryWordCount]uint64)(unsafe.Pointer(right))
	for word := 0; word < fixedEvictionEntryWordCount; word++ {
		delta := (leftWords[word] ^ rightWords[word]) & mask
		leftWords[word] ^= delta
		rightWords[word] ^= delta
	}
}

func fixedEvictionLessByTag(
	left *fixedEvictionEntry,
	right *fixedEvictionEntry,
) int {
	tagLess := ctLessUint64(left.tag, right.tag)
	tagEqual := utils.CtEqUint64(left.tag, right.tag)
	realBeforeSpecial := utils.CtNot(left.specialDummy) & right.specialDummy
	return tagLess | (tagEqual & realBeforeSpecial)
}

func fixedEvictionLessCountersFirst(
	left *fixedEvictionEntry,
	right *fixedEvictionEntry,
	overflowTag uint64,
) int {
	sameKind := utils.CtEqUint64(
		uint64(left.specialDummy&1),
		uint64(right.specialDummy&1),
	)
	tagLess := ctLessUint64(left.tag, right.tag)
	leftIsCounter := left.specialDummy & ctLessUint64(left.tag, overflowTag)
	rightIsOverflowDummy := right.specialDummy & utils.CtNot(
		ctLessUint64(right.tag, overflowTag),
	)

	return (sameKind & tagLess) |
		(leftIsCounter & utils.CtNot(right.specialDummy)) |
		(utils.CtNot(left.specialDummy) & rightIsOverflowDummy)
}

func fixedEvictionLess(
	left *fixedEvictionEntry,
	right *fixedEvictionEntry,
	sortKind int,
	overflowTag uint64,
) int {
	if sortKind == fixedEvictionSortCountersFirst {
		return fixedEvictionLessCountersFirst(left, right, overflowTag)
	}
	return fixedEvictionLessByTag(left, right)
}

func nextPowerOfTwo(value int) int {
	power := 1
	for power < value {
		power <<= 1
	}
	return power
}

// obliviousSortFixedEviction is the arbitrary-length bitonic network used by
// EnigMap's ObliSort(). Every compare/exchange address is a function only of
// the public work-buffer length; compare outcomes only feed branchless swaps.
func obliviousSortFixedEviction(
	entries []fixedEvictionEntry,
	sortKind int,
	overflowTag uint64,
) {
	bitonicSortFixedEviction(
		entries,
		0,
		len(entries),
		1,
		sortKind,
		overflowTag,
	)
}

func bitonicSortFixedEviction(
	entries []fixedEvictionEntry,
	low int,
	length int,
	direction int,
	sortKind int,
	overflowTag uint64,
) {
	if length <= 1 {
		return
	}

	middle := length >> 1
	bitonicSortFixedEviction(
		entries,
		low,
		middle,
		direction^1,
		sortKind,
		overflowTag,
	)
	bitonicSortFixedEviction(
		entries,
		low+middle,
		length-middle,
		direction,
		sortKind,
		overflowTag,
	)
	bitonicMergeFixedEviction(
		entries,
		low,
		length,
		direction,
		sortKind,
		overflowTag,
	)
}

func bitonicMergeFixedEviction(
	entries []fixedEvictionEntry,
	low int,
	length int,
	direction int,
	sortKind int,
	overflowTag uint64,
) {
	if length <= 1 {
		return
	}

	stride := nextPowerOfTwo(length) >> 1
	for index := low; index < low+length-stride; index++ {
		other := index + stride
		less := fixedEvictionLess(
			&entries[index],
			&entries[other],
			sortKind,
			overflowTag,
		)
		swap := direction ^ less
		conditionalSwapFixedEvictionEntries(
			&entries[index],
			&entries[other],
			swap,
		)
	}

	bitonicMergeFixedEviction(
		entries,
		low,
		stride,
		direction,
		sortKind,
		overflowTag,
	)
	bitonicMergeFixedEviction(
		entries,
		low+stride,
		length-stride,
		direction,
		sortKind,
		overflowTag,
	)
}

func fixedEvictionShallowness(
	block StashedBlock[NodeBlock[FixedValue]],
	leaf Leaf,
	pathLength int,
) uint64 {
	overflowTag := uint64(pathLength)
	tag := overflowTag
	found := 0

	for shallowness := 0; shallowness < pathLength; shallowness++ {
		match := utils.CtEqUint64(
			uint64(block.Block.Leaf)>>uint(shallowness),
			uint64(leaf)>>uint(shallowness),
		)
		take := match & utils.CtNot(found)
		tag = selectUint64(tag, uint64(shallowness), take)
		found |= match
	}

	eligible := block.Block.Valid & utils.CtNot(block.Sticky)
	return selectUint64(overflowTag, tag, eligible)
}

func (stash *Stash[T]) validBlockCount() int {
	count := 0
	for index := 0; index < len(stash.blocks); index++ {
		count += stash.blocks[index].Block.Valid & 1
	}
	return count
}

// EvictToPath fills a root-to-leaf path from the deepest bucket. FixedValue
// fixed-capacity stashes use the EnigMap sort-based eviction below; other
// stashes retain the generic greedy fallback. Sticky entries remain in the
// stash even when their position is on the path.
func (stash *Stash[T]) EvictToPath(
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[T] {
	return stash.EvictToPathInto(
		makeBucketPath[T](pathLength, bucketSize),
		leaf,
		pathLength,
		bucketSize,
	)
}

// EvictToPathInto is the allocation-free internal path used by Client. The
// caller supplies fixed public-size scratch storage; the eviction scan and sort
// schedule is identical to EvictToPath and does not depend on secret data.
func (stash *Stash[T]) EvictToPathInto(
	path []Bucket[T],
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[T] {
	if !preparePathScratch(path, pathLength, bucketSize) {
		return nil
	}
	path = path[:pathLength]
	if fixedStash, ok := any(stash).(*Stash[NodeBlock[FixedValue]]); ok && stash.fixedCapacity {
		fixedPath := evictFixedValueToPathInto(
			fixedStash,
			any(path).([]Bucket[NodeBlock[FixedValue]]),
			leaf,
			pathLength,
			bucketSize,
		)
		return any(fixedPath).([]Bucket[T])
	}

	var stackUsed [BucketSize]int
	used := stackUsed[:]
	if bucketSize > len(stackUsed) {
		// Generic callers may choose a nonstandard Z. The normal ENIGMAP path is
		// Z=BucketSize and therefore stays allocation-free.
		used = make([]int, bucketSize)
	} else {
		used = used[:bucketSize]
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

func preparePathScratch[T core.Selectable[T]](
	path []Bucket[T],
	pathLength int,
	bucketSize int,
) bool {
	if pathLength < 0 || bucketSize < 0 || len(path) < pathLength {
		return false
	}
	var zero Block[T]
	for depth := 0; depth < pathLength; depth++ {
		if len(path[depth]) != bucketSize {
			return false
		}
		for slot := 0; slot < bucketSize; slot++ {
			path[depth][slot] = zero
		}
	}
	return true
}

// evictFixedValueToPath follows EnigMap's sort-based Path ORAM eviction:
//
//  1. tag stash+path blocks by their deepest compatible path level;
//  2. obliviously sort by tag and count each exact-tag run;
//  3. obliviously sort the per-level counters to the front;
//  4. linearly add the exact amount of path padding required;
//  5. obliviously sort once more by tag;
//  6. write the first pathLength*bucketSize entries back to the path and
//     retain the fixed-capacity remainder as the stash.
//
// Sticky blocks receive the overflow tag, preserving the Go client's pinning
// semantics while keeping the same fixed sort/scan trace.
func evictFixedValueToPath(
	stash *Stash[NodeBlock[FixedValue]],
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[NodeBlock[FixedValue]] {
	return evictFixedValueToPathInto(
		stash,
		makeBucketPath[NodeBlock[FixedValue]](pathLength, bucketSize),
		leaf,
		pathLength,
		bucketSize,
	)
}

func evictFixedValueToPathInto(
	stash *Stash[NodeBlock[FixedValue]],
	path []Bucket[NodeBlock[FixedValue]],
	leaf Leaf,
	pathLength int,
	bucketSize int,
) []Bucket[NodeBlock[FixedValue]] {
	if !preparePathScratch(path, pathLength, bucketSize) {
		return nil
	}
	path = path[:pathLength]

	if pathLength == 0 || bucketSize == 0 {
		return path
	}

	overflowTag := uint64(pathLength) //pathLengthがスタッシュに置いておくことを示すタグ
	baseSize := len(stash.blocks)
	pathBlockCount := pathLength * bucketSize
	workLength := baseSize + pathBlockCount
	if workLength < baseSize {
		return path
	}

	var stackWork [fixedEvictionStackEntryCapacity]fixedEvictionEntry
	var work []fixedEvictionEntry
	if workLength <= len(stackWork) {
		work = stackWork[:workLength]
	} else {
		// Profile-only custom stash sizes can exceed the bounded stack scratch
		// area. Preserve their behavior without making the normal hot path pay
		// a heap allocation.
		work = make([]fixedEvictionEntry, workLength)
	}

	// 1) Tag the concatenated old-stash + complete-path work set. Invalid
	// entries go directly to the overflow tag; sticky entries are valid but are
	// deliberately made ineligible for this eviction by the same overflow tag.
	for index := 0; index < baseSize; index++ {
		entry := stash.blocks[index]
		invalid := utils.CtNot(entry.Block.Valid)
		work[index] = fixedEvictionEntry{
			stashed:      entry,
			tag:          fixedEvictionShallowness(entry, leaf, pathLength),
			specialDummy: invalid,
		}
	}

	// One special sentinel per path level. Sort #1 places each sentinel after
	// the real entries having the same exact deepest-compatible level.
	for level := 0; level < pathLength; level++ {
		work[baseSize+level] = fixedEvictionEntry{
			tag:          uint64(level),
			specialDummy: 1,
		}
	}
	originalSize := baseSize + pathLength

	// 2) Sort #1: exact-level runs, with the counter sentinel at each run end.
	obliviousSortFixedEviction(
		work[:originalSize],
		fixedEvictionSortByTag,
		overflowTag,
	)

	// 3) Linear count scan. The sentinel's otherwise-unused dummy block address
	// carries the exact number of real entries in that tag run into sort #2.
	count := uint64(0)
	currTag := uint64(0)
	for index := 0; index < originalSize; index++ {
		isEnd := work[index].specialDummy & ctLessUint64(currTag, overflowTag)
		work[index].stashed.Block.BlockAddress = BlockAddress(selectUint64(
			uint64(work[index].stashed.Block.BlockAddress),
			count,
			isEnd,
		))
		currTag = selectUint64(currTag, currTag+1, isEnd)
		count++
		count = selectUint64(count, 0, isEnd)
	}

	// 4) Sort #2: move the pathLength counter sentinels to work[0:pathLength]
	// in level order; ordinary real entries remain behind them.
	obliviousSortFixedEviction(
		work[:originalSize],
		fixedEvictionSortCountersFirst,
		overflowTag,
	)

	// 5) Linear padding scan. Maintain the cumulative number of entries that can
	// reach each leaf-to-root prefix. Whenever that prefix has fewer than its
	// public target slots, turn the counter sentinel plus up to Z-1 preallocated
	// work entries into path-padding dummies for this level.
	count = 0
	appendIndex := originalSize
	for level := 0; level < pathLength; level++ {
		counter := uint64(work[level].stashed.Block.BlockAddress)
		count += counter
		target := uint64((level + 1) * bucketSize)

		work[level].stashed.Block.BlockAddress = 0
		enough := utils.CtNot(ctLessUint64(count, target))
		work[level].tag = selectUint64(
			work[level].tag,
			overflowTag,
			enough,
		)

		needFirstPadding := ctLessUint64(count, target)
		count += uint64(needFirstPadding)

		for slot := 0; slot < bucketSize-1; slot++ {
			needPadding := ctLessUint64(count, target)
			work[appendIndex] = fixedEvictionEntry{
				tag: selectUint64(
					overflowTag,
					uint64(level),
					needPadding,
				),
				// These are ordinary path-padding dummies, matching EnigMap's
				// isDummy=false padding records when needed. Unused preallocated
				// records are marked special at the overflow tag so they stay
				// behind sticky real blocks in the fixed-capacity remainder.
				specialDummy: utils.CtNot(needPadding),
			}
			count += uint64(needPadding)
			appendIndex++
		}
	}

	// 6) Sort #3: first pathBlockCount entries are now exactly the leaf-to-root
	// path buckets; excess real blocks and sticky blocks follow as stash data.
	obliviousSortFixedEviction(
		work,
		fixedEvictionSortByTag,
		overflowTag,
	)

	for shallowness := 0; shallowness < pathLength; shallowness++ {
		depth := pathLength - 1 - shallowness
		for slot := 0; slot < bucketSize; slot++ {
			workIndex := shallowness*bucketSize + slot
			path[depth][slot] = work[workIndex].stashed.Block
		}
	}

	// Keep a fixed public stash size. The final tag sort places all real
	// overflow entries before special overflow dummies, so a straight linear
	// copy is sufficient; no per-block stash search/compaction is needed.
	capacity := stash.capacity
	if capacity <= 0 {
		capacity = baseSize - pathBlockCount
	}
	remainder := work[pathBlockCount:]
	copyCount := capacity
	if copyCount > len(remainder) {
		copyCount = len(remainder)
	}
	newStash := stash.blocks[:capacity]
	for index := 0; index < copyCount; index++ {
		newStash[index] = remainder[index].stashed
	}
	var zero StashedBlock[NodeBlock[FixedValue]]
	for index := copyCount; index < capacity; index++ {
		newStash[index] = zero
	}
	stash.blocks = newStash

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
	return stash.EvictToPathNonObliviousInto(
		makeBucketPath[T](pathLength, bucketSize),
		leaf,
		pathLength,
		bucketSize,
	)
}

func (stash *Stash[T]) EvictToPathNonObliviousInto(
	path []Bucket[T],
	leaf Leaf,
	pathLength int,
	bucketSize int,
) ([]Bucket[T], uint64) {
	if !preparePathScratch(path, pathLength, bucketSize) {
		return nil, 0
	}
	path = path[:pathLength]
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
	if stash.fixedCapacity {
		stash.compactFixedCapacityNonOblivious()
	} else {
		stash.compact()
	}
	return path, stashVisits
}

// compactFixedCapacityNonOblivious restores the fixed public stash length
// after the explicitly non-oblivious profiling eviction path. The optimized
// FixedValue AddPath temporarily extends stash.blocks with the complete path,
// so this branch must pack survivors before truncating back to capacity.
func (stash *Stash[T]) compactFixedCapacityNonOblivious() {
	capacity := stash.capacity
	if capacity <= 0 {
		return
	}

	if len(stash.blocks) < capacity {
		grown := make([]StashedBlock[T], capacity)
		copy(grown, stash.blocks)
		stash.blocks = grown
	}

	write := 0
	for read := 0; read < len(stash.blocks); read++ {
		if stash.blocks[read].Block.Valid == 0 {
			continue
		}
		if write < capacity {
			stash.blocks[write] = stash.blocks[read]
		}
		write++
	}

	if write > capacity {
		write = capacity
	}
	var zero StashedBlock[T]
	for index := write; index < capacity; index++ {
		stash.blocks[index] = zero
	}
	stash.blocks = stash.blocks[:capacity]
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
