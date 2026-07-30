package eoram

import (
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
)

func TestNodeBlockPreservesAVLNode(t *testing.T) {
	var value omap.Value
	copy(value[:], "value")

	node := avl.Node[omap.Value]{
		Key:     avl.NewKey(make([]byte, avl.KeySize)),
		Value:   value,
		Balance: avl.BalanceRight,
		Left:    11,
		Right:   22,
		Valid:   1,
	}

	got := newNodeBlock(node).avlNode()
	if got != node {
		t.Fatalf("avlNode() = %+v, want %+v", got, node)
	}
}

func TestNodeBlockSelect(t *testing.T) {
	current := avl.Node[omap.Value]{
		Key:     avl.NewKey(make([]byte, avl.KeySize)),
		Balance: avl.BalanceLeft,
		Left:    11,
		Right:   22,
		Valid:   1,
	}
	candidateKey := make([]byte, avl.KeySize)
	candidateKey[0] = 1
	candidate := avl.Node[omap.Value]{
		Key:     avl.NewKey(candidateKey),
		Balance: avl.BalanceRight,
		Left:    33,
		Right:   44,
	}
	copy(candidate.Value[:], "candidate")

	if got := newNodeBlock(current).Select(newNodeBlock(candidate), 0).avlNode(); got != current {
		t.Fatalf("Select(condition=0) = %+v, want %+v", got, current)
	}
	if got := newNodeBlock(current).Select(newNodeBlock(candidate), 1).avlNode(); got != candidate {
		t.Fatalf("Select(condition=1) = %+v, want %+v", got, candidate)
	}
}
