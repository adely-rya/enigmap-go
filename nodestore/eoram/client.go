package eoram

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
// V is the AVL value type stored in each ORAM block. B is the logical bucket
// type stored through OramServer.
type Client[V any, B any] struct {
	server OramServer[B]
	stash  Stash[V]

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}
