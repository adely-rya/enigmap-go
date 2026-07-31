package eoram

// Address identifies a logical ORAM block.
type Address uint64

// Position identifies a leaf in the ORAM tree.
type Position uint64

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
// B is the logical bucket type stored through OramServer. T is the stash entry
// type used by the ENIGMAP-specific stash implementation.
type Client[B any, T any] struct {
	server OramServer[B]
	stash  Stash[T]

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}
