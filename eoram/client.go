package eoram

// Address identifies a logical ORAM block.
type Address uint64

// Position identifies a leaf in the ORAM tree.
type Position uint64

// PathServer is the path-level storage boundary used by Client.
//
// oram/pathoram.ORAMServer satisfies this interface. Its underlying
// oram/core.IO can therefore be wrapped with memory, file, cache, or encrypted
// IO without changing Client.
type PathServer[B any] interface {
	LeafCount() uint64
	ReadPath(position uint64) ([]B, error)
	WritePath(path []B, position uint64) error
}

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
// B is the logical bucket type stored through PathServer. S is the
// ENIGMAP-specific stash type; keeping it separate lets the implementation add
// sticky blocks without modifying the reusable ORAM storage boundary.
type Client[B any, S any] struct {
	server PathServer[B]
	stash  S

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}
