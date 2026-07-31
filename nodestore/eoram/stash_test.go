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
		Address:  11,
		Position: 22,
		Valid:    1,
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
		blocks: make([]StashedBlock[selectableString], 4),
	}
	path := []Bucket[selectableString]{
		{
			{Address: 1, Data: "root", Valid: 1},
			{},
		},
		{
			{Address: 2, Data: "leaf", Valid: 1},
			{},
		},
	}

	if got := stash.AddPath(path, 2); got != 1 {
		t.Fatalf("AddPath = %d, want 1", got)
	}

	want := map[Address]selectableString{1: "root", 2: "leaf"}
	for _, entry := range stash.blocks {
		if entry.Block.Valid == 0 {
			continue
		}
		if entry.Block.Data != want[entry.Block.Address] {
			t.Fatalf(
				"stash block %d data = %q, want %q",
				entry.Block.Address,
				entry.Block.Data,
				want[entry.Block.Address],
			)
		}
		if entry.Sticky != 0 {
			t.Fatalf("path block %d is sticky", entry.Block.Address)
		}
		delete(want, entry.Block.Address)
	}
	if len(want) != 0 {
		t.Fatalf("blocks missing from stash: %v", want)
	}
}

func TestStashAddPathReportsOverflow(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: make([]StashedBlock[selectableString], 1),
	}
	path := []Bucket[selectableString]{
		{
			{Address: 1, Valid: 1},
			{Address: 2, Valid: 1},
		},
	}

	if got := stash.AddPath(path, 2); got != 0 {
		t.Fatalf("AddPath = %d, want 0", got)
	}
}

func TestStashAddPathRejectsBucketSizeMismatch(t *testing.T) {
	stash := Stash[selectableString]{
		blocks: make([]StashedBlock[selectableString], 2),
	}

	if got := stash.AddPath([]Bucket[selectableString]{{{}}}, 2); got != 0 {
		t.Fatalf("AddPath = %d, want 0", got)
	}
}
