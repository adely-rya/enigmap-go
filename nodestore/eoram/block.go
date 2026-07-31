package eoram

import "github.com/adely-rya/oram/core"

// Address identifies a logical ORAM block.
type Address uint64

// Position identifies a leaf in the ORAM tree.
type Position uint64

// Block is a logical Path ORAM block.
type Block[T core.Selectable[T]] struct {
	Address  Address
	Position Position
	Data     T
	Valid    int
}

type Bucket[T core.Selectable[T]] []Block[T]
