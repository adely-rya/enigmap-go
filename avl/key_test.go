package avl

import (
	"bytes"
	"strconv"
	"testing"
)

func TestKeyWordRepresentationPreservesLexicographicOrder(t *testing.T) {
	keys := [][]byte{
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 1},
		{0, 0, 0, 1, 0, 0, 0, 0},
		{0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0x80, 0, 0, 0, 0, 0, 0, 0},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}
	for _, leftBytes := range keys {
		left := NewKey(leftBytes)
		if got := left.Bytes(); !bytes.Equal(got, leftBytes) {
			t.Fatalf("round trip = %x, want %x", got, leftBytes)
		}
		for _, rightBytes := range keys {
			right := NewKey(rightBytes)
			got := CompareKey(left, right)
			want := bytes.Compare(leftBytes, rightBytes)
			if got != want {
				t.Fatalf("CompareKey(%x, %x) = %d, want %d", leftBytes, rightBytes, got, want)
			}
			if selected := SelectKey(left, right, 0); selected != left {
				t.Fatalf("SelectKey condition 0 selected %x, want %x", selected.Bytes(), leftBytes)
			}
			if selected := SelectKey(left, right, 1); selected != right {
				t.Fatalf("SelectKey condition 1 selected %x, want %x", selected.Bytes(), rightBytes)
			}
		}
	}
}

func TestNewKeyRejectsWrongLength(t *testing.T) {
	for _, size := range []int{KeySize - 1, KeySize + 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("NewKey accepted length %d", size)
				}
			}()
			NewKey(make([]byte, size))
		})
	}
}
