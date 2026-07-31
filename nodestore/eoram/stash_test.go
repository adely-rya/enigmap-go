package eoram

import "testing"

func TestBlockMakeStashedBlock(t *testing.T) {
	block := Block[string]{
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
	block := Block[string]{}

	if got := block.MakeStashedBlock(2).Sticky; got != 0 {
		t.Fatalf("MakeStashedBlock(2).Sticky = %d, want 0", got)
	}
	if got := block.MakeStashedBlock(3).Sticky; got != 1 {
		t.Fatalf("MakeStashedBlock(3).Sticky = %d, want 1", got)
	}
}
