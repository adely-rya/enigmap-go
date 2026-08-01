package eoram

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/oram/core"
)

const (
	defaultBucketSize = BucketSize
	securityBits      = 128
)

var ErrStashOverflow = errors.New("eoram: stash overflow")
var ErrBlockNotFound = errors.New("eoram: block not found")
var ErrAccessAlreadyOpen = errors.New("eoram: access already open")
var ErrNoAccessOpen = errors.New("eoram: no access open")
var ErrAccessTokenMismatch = errors.New("eoram: access token mismatch")

type accessPhase uint8

const (
	accessReady accessPhase = iota
	accessOpen
)

// accessState records the single BeginAccess operation awaiting FinishAccess.
type accessState struct {
	phase        accessPhase
	blockAddress BlockAddress
	oldLeaf      Leaf
	newLeaf      Leaf
	pathLength   int
}

// Client holds the state required by the ENIGMAP variant of Path ORAM.
//
// T is the selectable payload stored in each ORAM block.
type Client[T core.Selectable[T]] struct {
	server OramServer[Bucket[NodeBlock[T]], core.IO[Bucket[NodeBlock[T]]]]
	stash  Stash[NodeBlock[T]]
	random io.Reader

	bucketSize           int
	nextFreeBlock        BlockAddress
	access               accessState
	stats                ClientStats
	nonObliviousEviction bool
}

type ClientStats struct {
	Accesses          uint64
	ReadPathTime      time.Duration
	AddPathTime       time.Duration
	PrepareAccessTime time.Duration
	UpdateBlockTime   time.Duration
	EvictToPathTime   time.Duration
	WritePathTime     time.Duration
	UnpinTime         time.Duration
	MaxStashLength    uint64
	PathBucketsRead   uint64
	AddPathBlocks     uint64
	PrepareStashScans uint64
	UpdateStashScans  uint64
	EvictStashScans   uint64
	EvictBlockSelects uint64
	UnpinStashScans   uint64
}

// NewOramClient creates an ENIGMAP Path ORAM client backed by server.
func NewOramClient[T core.Selectable[T]](
	storage core.IO[Bucket[NodeBlock[T]]],
	bucketSize int,
) (*Client[T], error) {
	stashSize := int(math.Ceil(
		(float64(securityBits) + math.Log2(14.0)) /
			(-math.Log2(0.6002)),
	))
	return NewOramClientWithStashSize(storage, bucketSize, stashSize)
}

func NewOramClientWithStashSize[T core.Selectable[T]](
	storage core.IO[Bucket[NodeBlock[T]]],
	bucketSize int,
	stashSize int,
) (*Client[T], error) {
	if bucketSize <= 0 {
		bucketSize = defaultBucketSize
	}

	server, err := NewOramServer(storage)
	if err != nil {
		return nil, err
	}

	if stashSize <= 0 {
		return nil, fmt.Errorf("eoram: stash size must be positive: %d", stashSize)
	}

	client := &Client[T]{
		server: server,
		stash: Stash[NodeBlock[T]]{
			blocks:   make([]StashedBlock[NodeBlock[T]], 0, stashSize),
			capacity: stashSize,
		},
		random:        cryptoRand.Reader,
		bucketSize:    bucketSize,
		nextFreeBlock: 1,
		access: accessState{
			phase: accessReady,
		},
	}
	if err := client.initializeTree(); err != nil {
		return nil, err
	}

	return client, nil
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
func (client *Client[T]) BeginAccess(pointer avl.Pointer, newLeaf avl.ORAMPosition) (avl.Node[T], avl.AccessToken, error) {
	var zeroNode avl.Node[T]
	var zeroToken avl.AccessToken

	if client.access.phase != accessReady {
		return zeroNode, zeroToken, ErrAccessAlreadyOpen
	}

	oldLeaf := Leaf(pointer.Position)
	started := time.Now()
	path, err := client.server.ReadPath(oldLeaf)
	client.stats.ReadPathTime += time.Since(started)
	if err != nil {
		return zeroNode, zeroToken, err
	}

	started = time.Now()
	added := client.stash.AddPath(path, client.bucketSize)
	client.stats.AddPathTime += time.Since(started)
	client.stats.PathBucketsRead += uint64(len(path))
	client.stats.AddPathBlocks += uint64(len(path) * client.bucketSize)
	if uint64(len(client.stash.blocks)) > client.stats.MaxStashLength {
		client.stats.MaxStashLength = uint64(len(client.stash.blocks))
	}
	if added == 0 {
		return zeroNode, zeroToken, ErrStashOverflow
	}

	blockAddress := BlockAddress(pointer.Address)
	prepareStashLength := len(client.stash.blocks)
	started = time.Now()
	value, found := client.stash.PrepareAccess(
		blockAddress,
		Leaf(newLeaf),
	)
	client.stats.PrepareAccessTime += time.Since(started)
	// PrepareAccess performs one scan to select the value and another to
	// update its position and sticky bit.
	client.stats.PrepareStashScans += uint64(2 * prepareStashLength)

	if found == 0 {
		return zeroNode, zeroToken, fmt.Errorf(
			"%w: BeginAccess blockAddress=%d oldLeaf=%d",
			ErrBlockNotFound,
			blockAddress,
			oldLeaf,
		)
	}

	client.access = accessState{
		phase:        accessOpen,
		blockAddress: blockAddress,
		oldLeaf:      oldLeaf,
		newLeaf:      Leaf(newLeaf),
		pathLength:   len(path),
	}
	client.stats.Accesses++

	return value.Node, avl.AccessToken{Address: pointer.Address}, nil
}

// FinishAccess updates the AVL node opened by BeginAccess.
func (client *Client[T]) FinishAccess(token avl.AccessToken, node avl.Node[T]) error {
	//先にBeginAccessが呼ばれているかを検証
	if err := client.validateFinishAccess(token); err != nil {
		return err
	}

	updateStashLength := len(client.stash.blocks)
	started := time.Now()
	updated := client.stash.UpdateAccessedBlock(
		client.access.blockAddress,
		NodeBlock[T]{Node: node},
	)
	client.stats.UpdateBlockTime += time.Since(started)
	client.stats.UpdateStashScans += uint64(updateStashLength)
	if updated == 0 {
		return fmt.Errorf(
			"%w: FinishAccess blockAddress=%d",
			ErrBlockNotFound,
			client.access.blockAddress,
		)
	}

	started = time.Now()
	var path []Bucket[NodeBlock[T]]
	if client.nonObliviousEviction {
		var visits uint64
		path, visits = client.stash.EvictToPathNonOblivious(
			client.access.oldLeaf,
			client.access.pathLength,
			client.bucketSize,
		)
		client.stats.EvictStashScans += visits
	} else {
		evictStashLength := len(client.stash.blocks)
		client.stats.EvictStashScans += uint64(client.access.pathLength * evictStashLength)
		client.stats.EvictBlockSelects += uint64(
			client.access.pathLength * evictStashLength * client.bucketSize,
		)
		path = client.stash.EvictToPath(
			client.access.oldLeaf,
			client.access.pathLength,
			client.bucketSize,
		)
	}
	client.stats.EvictToPathTime += time.Since(started)
	client.stats.UnpinStashScans += uint64(len(client.stash.blocks))
	started = time.Now()
	if err := client.server.WritePath(path, client.access.oldLeaf); err != nil {
		client.stats.WritePathTime += time.Since(started)
		return err
	}
	client.stats.WritePathTime += time.Since(started)

	started = time.Now()
	unpinned := client.stash.Unpin(client.access.blockAddress)
	client.stats.UnpinTime += time.Since(started)
	if unpinned == 0 {
		return fmt.Errorf(
			"%w: FinishAccess unpin blockAddress=%d",
			ErrBlockNotFound,
			client.access.blockAddress,
		)
	}
	client.access = accessState{phase: accessReady}

	return nil
}

func (client *Client[T]) Stats() ClientStats { return client.stats }

// EnableNonObliviousEvictionForProfile enables the branchful C++ comparison
// mode. It is intentionally explicit because this weakens strong obliviousness.
func (client *Client[T]) EnableNonObliviousEvictionForProfile() {
	client.nonObliviousEviction = true
}

func (client *Client[T]) validateFinishAccess(token avl.AccessToken) error {
	if client.access.phase != accessOpen {
		return ErrNoAccessOpen
	}
	if BlockAddress(token.Address) != client.access.blockAddress {
		return ErrAccessTokenMismatch
	}

	return nil
}

// Allocate creates one pointer-addressed AVL node block.
func (client *Client[T]) Allocate(node avl.Node[T]) (avl.Pointer, error) {
	var zero avl.Pointer

	newLeaf, err := client.NewPosition()
	if err != nil {
		return zero, err
	}

	blockAddress := client.nextFreeBlock
	if client.stash.PutPinned(Block[NodeBlock[T]]{
		BlockAddress: blockAddress,
		Leaf:         Leaf(newLeaf),
		Data:         NodeBlock[T]{Node: node},
		Valid:        1,
	}) == 0 {
		return zero, ErrStashOverflow
	}

	client.nextFreeBlock++
	return avl.Pointer{
		Address:  avl.NodeID(blockAddress),
		Position: newLeaf,
	}, nil
}

// FreeMaybe conditionally releases an AVL node pointer.
func (client *Client[T]) FreeMaybe(
	pointer avl.Pointer,
	condition int,
) error {
	if client.stash.FreeMaybe(
		BlockAddress(pointer.Address),
		condition,
	) == 0 {
		return fmt.Errorf(
			"%w: FreeMaybe blockAddress=%d",
			ErrBlockNotFound,
			pointer.Address,
		)
	}

	return nil
}

// UncacheAll marks every sticky stash block as eligible for eviction.
func (client *Client[T]) UncacheAll() {
	client.stash.UnpinAll()
}
