package eoram

import "testing"

type selectableString string

func (value selectableString) Select(
	candidate selectableString,
	condition int,
) selectableString {
	if condition == 1 {
		return candidate
	}

	return value
}

func TestBlockMakeStashedBlock(t *testing.T) {
	block := Block[selectableString]{
		BlockAddress: 11,
		Leaf:         22,
		Valid:        1,
	}

	stashed := block.MakeStashedBlock(1)

	if stashed.Block != block {
		t.Fatalf("stashed block = %+v, want %+v", stashed.Block, block)
	}
	if stashed.Sticky != 1 {
		t.Fatalf("stashed Sticky = %d, want 1", stashed.Sticky)
	}
}

func TestBlockMakeStashedBlockNormalizesSticky(t *testing.T) {
	block := Block[selectableString]{}

	if got := block.MakeStashedBlock(2).Sticky; got != 0 {
		t.Fatalf("MakeStashedBlock(2).Sticky = %d, want 0", got)
	}
	if got := block.MakeStashedBlock(3).Sticky; got != 1 {
		t.Fatalf("MakeStashedBlock(3).Sticky = %d, want 1", got)
	}
}

func TestStashAddPath(t *testing.T) {
	stash := Stash[selectableString]{
		capacity: 4,
	}
	path := []Bucket[selectableString]{
		{
			{BlockAddress: 1, Data: "root", Valid: 1},
			{},
		},
		{
			{BlockAddress: 2, Data: "leaf", Valid: 1},
			{},
		},
	}

	if got := stash.AddPath(path, 2); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}

	want := map[BlockAddress]selectableString{1: "root", 2: "leaf"}
	for _, entry := range stash.blocks {
		if entry.Block.Valid == 0 {
			continue
		}
		if entry.Block.Data != want[entry.Block.BlockAddress] {
			t.Fatalf(
				"stash block %d data = %q, want %q",
				entry.Block.BlockAddress,
				entry.Block.Data,
				want[entry.Block.BlockAddress],
			)
		}
		if entry.Sticky != 0 {
			t.Fatalf("path block %d is sticky", entry.Block.BlockAddress)
		}
		delete(want, entry.Block.BlockAddress)
	}
	if len(want) != 0 {
		t.Fatalf("blocks missing from stash: %v", want)
	}
}

func TestStashAddPathReportsOverflow(t *testing.T) {
	stash := Stash[selectableString]{
		capacity: 1,
	}
	path := []Bucket[selectableString]{
		{
			{BlockAddress: 1, Valid: 1},
			{BlockAddress: 2, Valid: 1},
		},
	}

	if got := stash.AddPath(path, 2); got != 0 {
		t.Fatalf("AddPath = %d, want 0", got)
	}
}

func TestStashAddPathRejectsBucketSizeMismatch(t *testing.T) {
	stash := Stash[selectableString]{capacity: 2}

	if got := stash.AddPath([]Bucket[selectableString]{{{}}}, 2); got != 0 {
		t.Fatalf("AddPath = %d, want 0", got)
	}
}

func TestStashGetReturnsBlockData(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: []StashedBlock[selectableString]{
			Block[selectableString]{
				BlockAddress: 10,
				Data:         "first",
				Valid:        1,
			}.MakeStashedBlock(0),
			Block[selectableString]{
				BlockAddress: 20,
				Data:         "wanted",
				Valid:        1,
			}.MakeStashedBlock(1),
			{},
		},
	}

	got, found := stash.Get(20)
	if found != 1 {
		t.Fatalf("Get found = %d, want 1", found)
	}
	if got != "wanted" {
		t.Fatalf("Get data = %q, want %q", got, "wanted")
	}
}

func TestStashGetReturnsNotFound(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: []StashedBlock[selectableString]{
			Block[selectableString]{
				BlockAddress: 10,
				Data:         "stored",
				Valid:        1,
			}.MakeStashedBlock(0),
			{},
		},
	}

	got, found := stash.Get(99)
	if found != 0 {
		t.Fatalf("Get found = %d, want 0", found)
	}
	if got != "" {
		t.Fatalf("Get data = %q, want zero value", got)
	}
}

func TestStashEvictToPathSkipsStickyBlocks(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: []StashedBlock[selectableString]{
			Block[selectableString]{
				BlockAddress: 1,
				Leaf:         2,
				Data:         "evictable",
				Valid:        1,
			}.MakeStashedBlock(0),
			Block[selectableString]{
				BlockAddress: 2,
				Leaf:         2,
				Data:         "sticky",
				Valid:        1,
			}.MakeStashedBlock(1),
		},
	}

	path := stash.EvictToPath(2, 3, 1)

	if path[2][0].BlockAddress != 1 || path[2][0].Data != "evictable" {
		t.Fatalf("leaf bucket = %+v, want evictable block", path[2][0])
	}
	if len(stash.blocks) != 1 {
		t.Fatalf("stash length = %d, want 1 sticky block", len(stash.blocks))
	}
	if stash.blocks[0].Block.Valid != 1 || stash.blocks[0].Sticky != 1 {
		t.Fatalf("sticky block changed: %+v", stash.blocks[0])
	}
}

func TestStashEvictToPathOnlyPlacesCompatibleBlocks(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: []StashedBlock[selectableString]{
			Block[selectableString]{
				BlockAddress: 1,
				Leaf:         2,
				Data:         "same leaf",
				Valid:        1,
			}.MakeStashedBlock(0),
			Block[selectableString]{
				BlockAddress: 2,
				Leaf:         0,
				Data:         "other branch",
				Valid:        1,
			}.MakeStashedBlock(0),
		},
	}

	path := stash.EvictToPath(2, 3, 1)

	if path[2][0].BlockAddress != 1 {
		t.Fatalf("leaf block address = %d, want 1", path[2][0].BlockAddress)
	}
	if path[1][0].Valid != 0 {
		t.Fatalf("incompatible block was placed below root: %+v", path[1][0])
	}
	if path[0][0].BlockAddress != 2 {
		t.Fatalf("root block address = %d, want 2", path[0][0].BlockAddress)
	}
}

func TestCtLessUint64(t *testing.T) {
	max := ^uint64(0)
	tests := []struct {
		left  uint64
		right uint64
		want  int
	}{
		{left: 0, right: 0, want: 0},
		{left: 0, right: 1, want: 1},
		{left: 1, right: 0, want: 0},
		{left: max, right: 0, want: 0},
		{left: 0, right: max, want: 1},
		{left: max - 1, right: max, want: 1},
		{left: max, right: max - 1, want: 0},
	}

	for _, test := range tests {
		if got := ctLessUint64(test.left, test.right); got != test.want {
			t.Fatalf(
				"ctLessUint64(%d, %d) = %d, want %d",
				test.left,
				test.right,
				got,
				test.want,
			)
		}
	}
}

func TestFixedValueAddPathConcatenatesWorkBuffer(t *testing.T) {
	stash := Stash[NodeBlock[FixedValue]]{
		blocks: []StashedBlock[NodeBlock[FixedValue]]{
			Block[NodeBlock[FixedValue]]{
				BlockAddress: 1,
				Leaf:         0,
				Valid:        1,
			}.MakeStashedBlock(1),
			{},
			{},
		},
		capacity:      3,
		fixedCapacity: true,
	}
	path := []Bucket[NodeBlock[FixedValue]]{
		{
			{BlockAddress: 10, Leaf: 0, Valid: 1},
			{BlockAddress: 11, Leaf: 1, Valid: 1},
		},
		{
			{BlockAddress: 12, Leaf: 2, Valid: 1},
			{},
		},
	}

	if got := stash.AddPath(path, 2); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}
	if got, want := len(stash.blocks), 3+len(path)*2; got != want {
		t.Fatalf("work buffer length = %d, want %d", got, want)
	}
	if stash.blocks[0].Block.BlockAddress != 1 || stash.blocks[0].Sticky != 1 {
		t.Fatalf("old stash prefix changed: %+v", stash.blocks[0])
	}

	wantAddresses := []BlockAddress{10, 11, 12, 0}
	for index, want := range wantAddresses {
		entry := stash.blocks[stash.capacity+index]
		if entry.Block.BlockAddress != want {
			t.Fatalf(
				"path work entry %d address = %d, want %d",
				index,
				entry.Block.BlockAddress,
				want,
			)
		}
		if entry.Sticky != 0 {
			t.Fatalf("path work entry %d Sticky = %d, want 0", index, entry.Sticky)
		}
	}
}

func TestFixedValueAddPathReusesReservedBackingBuffer(t *testing.T) {
	const (
		stashCapacity = 3
		bucketSize    = 2
	)
	path := []Bucket[NodeBlock[FixedValue]]{
		{
			{BlockAddress: 10, Leaf: 0, Valid: 1},
			{BlockAddress: 11, Leaf: 1, Valid: 1},
		},
		{
			{BlockAddress: 12, Leaf: 2, Valid: 1},
			{},
		},
	}
	blocks := make(
		[]StashedBlock[NodeBlock[FixedValue]],
		stashCapacity,
		stashCapacity+len(path)*bucketSize,
	)
	stash := Stash[NodeBlock[FixedValue]]{
		blocks:        blocks,
		capacity:      stashCapacity,
		fixedCapacity: true,
	}
	backingStart := &stash.blocks[0]

	if got := stash.AddPath(path, bucketSize); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}
	if &stash.blocks[0] != backingStart {
		t.Fatal("AddPath replaced the reserved stash backing buffer")
	}
}

func TestFixedValueSortEvictionPreservesBlocksAndSticky(t *testing.T) {
	const (
		capacity   = 5
		pathLength = 3
		bucketSize = 1
		targetLeaf = Leaf(2)
	)

	stash := Stash[NodeBlock[FixedValue]]{
		blocks: []StashedBlock[NodeBlock[FixedValue]]{
			Block[NodeBlock[FixedValue]]{
				BlockAddress: 1,
				Leaf:         2,
				Valid:        1,
			}.MakeStashedBlock(1),
			Block[NodeBlock[FixedValue]]{
				BlockAddress: 2,
				Leaf:         2,
				Valid:        1,
			}.MakeStashedBlock(0),
			Block[NodeBlock[FixedValue]]{
				BlockAddress: 3,
				Leaf:         0,
				Valid:        1,
			}.MakeStashedBlock(0),
			{},
			{},
		},
		capacity:      capacity,
		fixedCapacity: true,
	}
	pathIn := []Bucket[NodeBlock[FixedValue]]{
		{{BlockAddress: 4, Leaf: 2, Valid: 1}},
		{{BlockAddress: 5, Leaf: 3, Valid: 1}},
		{{}},
	}
	if got := stash.AddPath(pathIn, bucketSize); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}

	path := stash.EvictToPath(targetLeaf, pathLength, bucketSize)
	if len(stash.blocks) != capacity {
		t.Fatalf("stash length after eviction = %d, want %d", len(stash.blocks), capacity)
	}

	seen := make(map[BlockAddress]int)
	validPathBlocks := 0
	for depth := 0; depth < pathLength; depth++ {
		block := path[depth][0]
		if block.Valid == 0 {
			continue
		}
		validPathBlocks++
		if block.BlockAddress == 1 {
			t.Fatalf("sticky block was evicted to depth %d", depth)
		}

		shift := uint(pathLength - 1 - depth)
		if depth != 0 && uint64(block.Leaf)>>shift != uint64(targetLeaf)>>shift {
			t.Fatalf(
				"block %d with leaf %d is incompatible with path depth %d",
				block.BlockAddress,
				block.Leaf,
				depth,
			)
		}
		seen[block.BlockAddress]++
	}
	if validPathBlocks != pathLength*bucketSize {
		t.Fatalf("valid path blocks = %d, want %d", validPathBlocks, pathLength*bucketSize)
	}

	stickyFound := false
	for _, entry := range stash.blocks {
		if entry.Block.Valid == 0 {
			continue
		}
		seen[entry.Block.BlockAddress]++
		if entry.Block.BlockAddress == 1 {
			stickyFound = true
			if entry.Sticky != 1 {
				t.Fatalf("sticky block Sticky = %d, want 1", entry.Sticky)
			}
		}
	}
	if !stickyFound {
		t.Fatal("sticky block missing from final stash")
	}

	for _, address := range []BlockAddress{1, 2, 3, 4, 5} {
		if seen[address] != 1 {
			t.Fatalf("block %d occurrence count = %d, want 1", address, seen[address])
		}
	}
}

func TestFixedValueNonObliviousEvictionRestoresFixedCapacity(t *testing.T) {
	stash := Stash[NodeBlock[FixedValue]]{
		blocks: []StashedBlock[NodeBlock[FixedValue]]{
			Block[NodeBlock[FixedValue]]{
				BlockAddress: 1,
				Leaf:         0,
				Valid:        1,
			}.MakeStashedBlock(1),
			{},
		},
		capacity:      2,
		fixedCapacity: true,
	}
	pathIn := []Bucket[NodeBlock[FixedValue]]{
		{{BlockAddress: 2, Leaf: 0, Valid: 1}},
	}
	if got := stash.AddPath(pathIn, 1); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}
	if len(stash.blocks) != 3 {
		t.Fatalf("work buffer length = %d, want 3", len(stash.blocks))
	}

	_, _ = stash.EvictToPathNonOblivious(0, 1, 1)
	if len(stash.blocks) != stash.capacity {
		t.Fatalf(
			"stash length after non-oblivious eviction = %d, want %d",
			len(stash.blocks),
			stash.capacity,
		)
	}
}
