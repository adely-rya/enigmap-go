package eoram

import "unsafe"

type fixedValueBlock = Block[NodeBlock[FixedValue]]

const fixedValueBlockWordCount = int(unsafe.Sizeof(fixedValueBlock{})) / 8

// selectFixedValueBlockWords selects every word of the concrete pointer-free
// block, including padding. This preserves a fixed memory trace but depends on
// the current Go struct layout, which is guarded by tests.
func selectFixedValueBlockWords(
	current fixedValueBlock,
	candidate fixedValueBlock,
	condition int,
) fixedValueBlock {
	var selected fixedValueBlock
	currentWords := (*[fixedValueBlockWordCount]uint64)(unsafe.Pointer(&current))
	candidateWords := (*[fixedValueBlockWordCount]uint64)(unsafe.Pointer(&candidate))
	selectedWords := (*[fixedValueBlockWordCount]uint64)(unsafe.Pointer(&selected))
	for word := 0; word < fixedValueBlockWordCount; word++ {
		selectedWords[word] = selectUint64(
			currentWords[word],
			candidateWords[word],
			condition,
		)
	}
	return selected
}
