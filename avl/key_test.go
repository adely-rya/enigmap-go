package avl

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCompareKey(t *testing.T) {
	key := func(prefix ...byte) []byte {
		result := make([]byte, KeySize)
		copy(result, prefix)
		return result
	}

	tests := []struct {
		name string
		a    []byte
		b    []byte
		want int
	}{
		{name: "equal", a: key(0x01), b: key(0x01), want: 0},
		{name: "less at first byte", a: key(0x01), b: key(0x02), want: -1},
		{name: "greater at first byte", a: key(0x02), b: key(0x01), want: 1},
		{
			name: "less at last byte",
			a:    bytes.Repeat([]byte{0x01}, KeySize),
			b:    append(bytes.Repeat([]byte{0x01}, KeySize-1), 0x02),
			want: -1,
		},
		{name: "embedded zero", a: key(0x01, 0x00, 0x01), b: key(0x01, 0x00, 0x02), want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewKey(tt.a)
			b := NewKey(tt.b)

			if got := CompareKey(a, b); got != tt.want {
				t.Fatalf("CompareKey() = %d, want %d", got, tt.want)
			}
			if got := CompareKey(b, a); got != -tt.want {
				t.Fatalf("reverse CompareKey() = %d, want %d", got, -tt.want)
			}
		})
	}
}

func TestNewKeyPanicsForInvalidLength(t *testing.T) {
	for _, size := range []int{0, KeySize - 1, KeySize + 1} {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("NewKey() did not panic for length %d", size)
				}
			}()

			NewKey(make([]byte, size))
		})
	}
}

func TestKeyCopiesInput(t *testing.T) {
	src := bytes.Repeat([]byte{0x01}, KeySize)
	key := NewKey(src)

	src[0] = 0xff
	if got := key.Bytes(); got[0] != 0x01 {
		t.Fatalf("Key.Bytes()[0] = %x, want 01", got[0])
	}
	if got := key.Len(); got != KeySize {
		t.Fatalf("Key.Len() = %d, want %d", got, KeySize)
	}
}
