package eoram

import (
	"math"

	"github.com/adely-rya/oram/core"
	"github.com/adely-rya/oram/oram/pathoram"
)

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

// NewOramClient creates an ENIGMAP Path ORAM client backed by server.
func NewOramClient[T core.Selectable[T]](
	storage core.IO[Bucket[T]],
	bucketSize int,
) *Client[T] {
	if bucketSize <= 0 {
		bucketSize = pathoram.BUCKET_SIZE
	}

	stashSize := int(math.Ceil(
		(float64(pathoram.SECURITY_BITS) + math.Log2(14.0)) /
			(-math.Log2(0.6002)),
	))

	return &Client[T]{
		server:     NewOramServer(storage),
		stash:      Stash[T]{blocks: make([]StashedBlock[T], stashSize)},
		bucketSize: bucketSize,
		access: accessState{
			phase: accessReady,
		},
	}
}
