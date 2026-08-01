package eoram

import (
	"encoding/binary"
	"fmt"

	"github.com/adely-rya/enigmap-go/config"
	"github.com/adely-rya/oram/core"
)

// FixedValue is the concrete eight-byte ORAM value used by the optimized
// fixed-layout path. Its byte representation remains stable for storage.
type FixedValue [config.ValueSize]byte

// Keep the specialized representation honest if the centralized setting is
// changed: both declarations compile only when ValueSize == one uint64.
var (
	_ [config.ValueSize - 8]struct{}
	_ [8 - config.ValueSize]struct{}
)

func (value FixedValue) Select(candidate FixedValue, condition int) FixedValue {
	currentWord := binary.LittleEndian.Uint64(value[:])
	candidateWord := binary.LittleEndian.Uint64(candidate[:])
	selectedWord := selectUint64(currentWord, candidateWord, condition)
	var selected FixedValue
	binary.LittleEndian.PutUint64(selected[:], selectedWord)
	return selected
}

func (value FixedValue) Serialize() ([]byte, error) {
	encoded := make([]byte, len(value))
	copy(encoded, value[:])
	return encoded, nil
}

func (FixedValue) Deserialize(encoded []byte) (core.Serializable, error) {
	if len(encoded) != config.ValueSize {
		return nil, fmt.Errorf(
			"eoram: fixed value size mismatch: got=%d want=%d",
			len(encoded),
			config.ValueSize,
		)
	}
	var value FixedValue
	copy(value[:], encoded)
	return value, nil
}

func selectUint64(current, candidate uint64, condition int) uint64 {
	mask := uint64(0) - uint64(condition&1)
	return current ^ ((current ^ candidate) & mask)
}
