package avl

import (
	"encoding/binary"
	"math/bits"

	"github.com/adely-rya/enigmap-go/config"
	"github.com/adely-rya/oram/utils"
)

// KeySize is kept as an AVL-facing alias for compatibility. The shared
// compile-time setting lives in package config.
const KeySize = config.KeySize

const keyWordCount = KeySize / 8

// Both array lengths compile only when KeySize is word aligned.
var (
	_ [KeySize % 8]struct{}
	_ [-(KeySize % 8)]struct{}
)

type Key struct {
	words [keyWordCount]uint64
}

func NewKey(src []byte) Key {
	if len(src) != KeySize {
		panic("key length must equal KeySize")
	}

	var key Key
	for index := range key.words {
		start := index * 8
		key.words[index] = binary.BigEndian.Uint64(src[start : start+8])
	}
	return key
}

func (k Key) Bytes() []byte {
	dst := make([]byte, KeySize)
	k.CopyBytes(dst)
	return dst
}

// CopyBytes writes the fixed-size key representation into dst without
// allocating. Bytes remains available for callers that need an owned slice.
func (k Key) CopyBytes(dst []byte) {
	if len(dst) != KeySize {
		panic("key destination length must equal KeySize")
	}
	for index, word := range k.words {
		start := index * 8
		binary.BigEndian.PutUint64(dst[start:start+8], word)
	}
}

func (k Key) Len() int {
	return KeySize
}

// SelectKey returns candidate when condition is 1 and current otherwise.
// It operates on the fixed arrays directly so hot ORAM loops do not allocate
// temporary byte slices for every conditional block move.
func SelectKey(current, candidate Key, condition int) Key {
	mask := uint64(0) - uint64(condition&1)
	var selected Key
	for index := range selected.words {
		selected.words[index] = current.words[index] ^
			((current.words[index] ^ candidate.words[index]) & mask)
	}
	return selected
}

// CompareKey scans every big-endian word and preserves the first difference.
func CompareKey(a, b Key) int {
	less := 0
	greater := 0
	equalSoFar := 1
	for index := range a.words {
		_, wordLess := bits.Sub64(a.words[index], b.words[index], 0)
		_, wordGreater := bits.Sub64(b.words[index], a.words[index], 0)
		less |= equalSoFar & int(wordLess)
		greater |= equalSoFar & int(wordGreater)
		equalSoFar &= utils.CtEqUint64(a.words[index], b.words[index])
	}
	return greater - less
}

// a < b  less = 1 greater = 0  return -1
// a = b  less = 0 greater = 0  return 0
// a > b  less = 0 greater = 1  return 1
