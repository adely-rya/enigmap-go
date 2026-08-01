package avl

import (
	"encoding/binary"
	"math/bits"

	"github.com/adely-rya/enigmap-go/config"
)

// KeySize is kept as an AVL-facing alias for compatibility. The shared
// compile-time setting lives in package config.
const KeySize = config.KeySize

// The word representation is intentionally available only for the configured
// eight-byte experiment. Both array lengths compile only when KeySize == 8.
var (
	_ [KeySize - 8]struct{}
	_ [8 - KeySize]struct{}
)

type Key struct {
	// Big-endian encoding makes unsigned word order equal byte-lexicographic
	// order while allowing one word-sized conditional move.
	word uint64
}

func NewKey(src []byte) Key {
	if len(src) != KeySize {
		panic("key length must equal KeySize")
	}

	return Key{word: binary.BigEndian.Uint64(src)}
}

func (k Key) Bytes() []byte {
	dst := make([]byte, KeySize)
	binary.BigEndian.PutUint64(dst, k.word)
	return dst
}

func (k Key) Len() int {
	return KeySize
}

// SelectKey returns candidate when condition is 1 and current otherwise.
// It operates on the fixed arrays directly so hot ORAM loops do not allocate
// temporary byte slices for every conditional block move.
func SelectKey(current, candidate Key, condition int) Key {
	mask := uint64(0) - uint64(condition&1)
	return Key{word: current.word ^ ((current.word ^ candidate.word) & mask)}
}

// CompareKey compares the big-endian words in byte-lexicographic order.
// bits.Sub64 returns the unsigned borrow without data-dependent branches.
func CompareKey(a, b Key) int {
	_, less := bits.Sub64(a.word, b.word, 0)
	_, greater := bits.Sub64(b.word, a.word, 0)
	return int(greater) - int(less)
}

// a < b  less = 1 greater = 0  return -1
// a = b  less = 0 greater = 0  return 0
// a > b  less = 0 greater = 1  return 1
