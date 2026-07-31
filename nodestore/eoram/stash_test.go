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
