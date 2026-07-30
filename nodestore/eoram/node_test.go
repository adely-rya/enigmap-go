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
