package eoram

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/core"
)

const (
	defaultBucketSize = 5
	securityBits      = 128
)

var ErrStashOverflow = errors.New("eoram: stash overflow")
var ErrBeginAccessIncomplete = errors.New(
	"eoram: BeginAccess block lookup is not implemented",
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
	server OramServer[Bucket[NodeBlock[T]], core.IO[Bucket[NodeBlock[T]]]]
	stash  Stash[NodeBlock[T]]
	random io.Reader

	bucketSize    int
	nextFreeBlock Address
	access        accessState
}

// NewOramClient creates an ENIGMAP Path ORAM client backed by server.
func NewOramClient[T core.Selectable[T]](
	storage core.IO[Bucket[NodeBlock[T]]],
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
		server: server,
		stash: Stash[NodeBlock[T]]{
			blocks: make([]StashedBlock[NodeBlock[T]], stashSize),
		},
		random:     cryptoRand.Reader,
		bucketSize: bucketSize,
		access: accessState{
			phase: accessReady,
		},
	}, nil
}

// ランダムにリーフを選定する
func (client *Client[T]) NewPosition() (avl.ORAMPosition, error) {
	var randomBytes [8]byte
	if _, err := io.ReadFull(client.random, randomBytes[:]); err != nil {
		return 0, err
	}

	randomValue := binary.LittleEndian.Uint64(randomBytes[:])
	return avl.ORAMPosition(
		randomValue & (client.server.LeafCount() - 1),
	), nil
}

// BeginAccess starts one AVL node access.
func (client *Client[T]) BeginAccess(
	pointer avl.Pointer,
	newPosition avl.ORAMPosition,
) (avl.Node[T], avl.AccessToken, error) {
	var zerovalue avl.Node[T]
	var zeroID avl.AccessToken

	leaf := Position(pointer.Position)
	path, err := client.server.ReadPath(leaf)
	if err != nil {
		return zerovalue, zeroID, err
	}

	if client.stash.AddPath(path, client.bucketSize) == 0 {
		return zerovalue, zeroID, ErrStashOverflow
	}

	return zerovalue, zeroID, ErrBeginAccessIncomplete
}

// FinishAccess updates the AVL node opened by BeginAccess.
func (client *Client[T]) FinishAccess(
	token avl.AccessToken,
	node avl.Node[T],
) error {
	panic("eoram: Client.FinishAccess is not implemented")
}

// Allocate creates one pointer-addressed AVL node block.
func (client *Client[T]) Allocate(
	node avl.Node[T],
) (avl.Pointer, error) {
	panic("eoram: Client.Allocate is not implemented")
}

// FreeMaybe conditionally releases an AVL node pointer.
func (client *Client[T]) FreeMaybe(
	pointer avl.Pointer,
	condition int,
) error {
	panic("eoram: Client.FreeMaybe is not implemented")
}

// UncacheAll marks every sticky stash block as eligible for eviction.
func (client *Client[T]) UncacheAll() {
	panic("eoram: Client.UncacheAll is not implemented")
}

func assertNodeStore[T core.Selectable[T]]() {
	var _ avl.NodeStore[T] = (*Client[T])(nil)
}
