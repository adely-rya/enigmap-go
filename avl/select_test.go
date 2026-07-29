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

func TestIsNotNullAddress(t *testing.T) {
	if got := isNotNullAddress(NullAddress); got != 0 {
		t.Fatalf("isNotNullAddress(NullAddress) = %d, want 0", got)
	}
	if got := isNotNullAddress(1); got != 1 {
		t.Fatalf("isNotNullAddress(1) = %d, want 1", got)
	}
	if got := isNotNullAddress(^NodeID(0)); got != 1 {
		t.Fatalf("isNotNullAddress(max) = %d, want 1", got)
	}
}

func TestNodeIDsEqual(t *testing.T) {
	if got := nodeIDsEqual(11, 11); got != 1 {
		t.Fatalf("nodeIDsEqual(same) = %d, want 1", got)
	}
	if got := nodeIDsEqual(11, 22); got != 0 {
		t.Fatalf("nodeIDsEqual(different) = %d, want 0", got)
	}
}
