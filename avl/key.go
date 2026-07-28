package avl

import (
	"crypto/subtle"
)

const KeySize = 33 //鍵サイズは固定長

type Key struct {
	data [KeySize]byte
}

func NewKey(src []byte) Key {
	if len(src) != KeySize {
		panic("key length must equal KeySize")
	}

	var key Key
	copy(key.data[:], src)
	return key
}

func (k Key) Bytes() []byte {
	dst := make([]byte, KeySize)
	copy(dst, k.data[:])
	return dst
}

func (k Key) Len() int {
	return KeySize
}

// CompareKey compares two keys lexicographically. It examines all KeySize
// bytes regardless of where the first difference occurs.
func CompareKey(a, b Key) int {
	less := 0
	greater := 0
	equalSoFar := 1

	for i := 0; i < KeySize; i++ {
		equal := subtle.ConstantTimeByteEq(a.data[i], b.data[i])
		aLess := subtle.ConstantTimeLessOrEq(
			int(a.data[i]),
			int(b.data[i]),
		) & (equal ^ 1)
		aGreater := subtle.ConstantTimeLessOrEq(
			int(b.data[i]),
			int(a.data[i]),
		) & (equal ^ 1)

		less |= equalSoFar & aLess
		greater |= equalSoFar & aGreater
		equalSoFar &= equal //一つでもゼロだったらずっとゼロ
	}

	return greater - less
}

// a < b  less = 1 greater = 0  return -1
// a = b  less = 0 greater = 0  return 0
// a > b  less = 0 greater = 1  return 1
