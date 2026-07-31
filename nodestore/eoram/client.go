package eoram

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"io"
	"math"

	"github.com/adely-rya/oram/core"
)

const (
	defaultBucketSize = 5
	securityBits      = 128
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
	random io.Reader

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}

// NewOramClient creates an ENIGMAP Path ORAM client backed by server.
func NewOramClient[T core.Selectable[T]](
	storage core.IO[Bucket[T]],
	bucketSize int,
) (*Client[T], error) {
	if bucketSize <= 0 {
		bucketSize = defaultBucketSize
	}

	server, err := NewOramServer(storage)
	if err != nil {
		return nil, err
	}

	stashSize := int(math.Ceil(
		(float64(securityBits) + math.Log2(14.0)) /
			(-math.Log2(0.6002)),
	))

	return &Client[T]{
		server:     server,
		stash:      Stash[T]{blocks: make([]StashedBlock[T], stashSize)},
		random:     cryptoRand.Reader,
		bucketSize: bucketSize,
		access: accessState{
			phase: accessReady,
		},
	}, nil
}

// NewPosition generates a new leaf position for an ORAM block.
func (client *Client[T]) NewPosition() (Position, error) {
	var randomBytes [8]byte
	if _, err := io.ReadFull(client.random, randomBytes[:]); err != nil {
		return 0, err
	}

	randomValue := binary.LittleEndian.Uint64(randomBytes[:])
	return Position(randomValue & (client.server.LeafCount() - 1)), nil
}

// BeginAccess reads the path at oldPosition into the stash and moves the
// requested block to newPosition. sticky controls whether eviction must retain
// the block in the stash.
func (client *Client[T]) BeginAccess(
	address Address,
	oldPosition Position,
	newPosition Position,
	sticky int,
) (T, error) {
	panic("eoram: Client.BeginAccess is not implemented")
}

// FinishAccess updates the accessed stash block and evicts the path opened by
// BeginAccess. markUncached controls whether the block becomes evictable.
func (client *Client[T]) FinishAccess(
	address Address,
	data T,
	markUncached int,
) error {
	panic("eoram: Client.FinishAccess is not implemented")
}

// Allocate places a new logical block in the stash at position.
func (client *Client[T]) Allocate(
	data T,
	position Position,
) (Address, error) {
	panic("eoram: Client.Allocate is not implemented")
}

// FreeMaybe releases address when condition is 1.
func (client *Client[T]) FreeMaybe(address Address, condition int) error {
	panic("eoram: Client.FreeMaybe is not implemented")
}

// UncacheAll marks every sticky stash block as eligible for eviction.
func (client *Client[T]) UncacheAll() {
	panic("eoram: Client.UncacheAll is not implemented")
}
