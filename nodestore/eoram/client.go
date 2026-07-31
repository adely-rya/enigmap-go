package eoram

import "github.com/adely-rya/oram/core"

type accessPhase uint8

const (
	accessReady accessPhase = iota
	accessOpen
)

// accessState records the single BeginAccess operation awaiting FinishAccess.
type accessState struct {
	phase       accessPhase
	address     Address
	oldPosition Position
	newPosition Position
}

// Client holds the state required by the ENIGMAP variant of Path ORAM.
//
// T is the selectable payload stored in each ORAM block.
type Client[T core.Selectable[T]] struct {
	server OramServer[Bucket[T], core.IO[Bucket[T]]]
	stash  Stash[T]

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}
