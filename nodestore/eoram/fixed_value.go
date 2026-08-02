package eoram

import (
	"fmt"

	"github.com/adely-rya/enigmap-go/config"
	"github.com/adely-rya/oram/core"
)

// FixedValue is the concrete fixed-size ORAM value used by the optimized
// fixed-layout path. Its byte representation remains stable for storage.
type FixedValue [config.ValueSize]byte

func (value FixedValue) Select(candidate FixedValue, condition int) FixedValue {
	var selected FixedValue
	mask := byte(0) - byte(condition&1)
	for index := range selected {
		selected[index] = value[index] ^ ((value[index] ^ candidate[index]) & mask)
	}
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
