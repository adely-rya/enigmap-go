package avl

import "testing"

func TestSelectNodeID(t *testing.T) {
	if got := selectNodeID(11, 22, 0); got != 11 {
		t.Fatalf("selectNodeID condition 0 = %d, want 11", got)
	}
	if got := selectNodeID(11, 22, 1); got != 22 {
		t.Fatalf("selectNodeID condition 1 = %d, want 22", got)
	}
}

func TestNodeIDPresent(t *testing.T) {
	if got := nodeIDPresent(NullAddress); got != 0 {
		t.Fatalf("nodeIDPresent(NullAddress) = %d, want 0", got)
	}
	if got := nodeIDPresent(1); got != 1 {
		t.Fatalf("nodeIDPresent(1) = %d, want 1", got)
	}
	if got := nodeIDPresent(^NodeID(0)); got != 1 {
		t.Fatalf("nodeIDPresent(max) = %d, want 1", got)
	}
}
