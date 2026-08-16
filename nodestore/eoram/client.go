package eoram

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
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
	server      OramServer[Bucket[NodeBlock[T]], core.IO[Bucket[NodeBlock[T]]]]
	stash       Stash[NodeBlock[T]]
	random      io.Reader
	randomBytes [8]byte
	pathScratch []Bucket[NodeBlock[T]]

	bucketSize                 int
	nextFreeBlock              BlockAddress
	access                     accessState
	stats                      ClientStats
	nonObliviousEviction       bool
	profileSequentialPositions bool
	profileNextPosition        uint64
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
	NewPositionTime   time.Duration
	NewPositions      uint64
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

	// The FixedValue path temporarily appends one complete ORAM path to the
	// fixed stash between BeginAccess and FinishAccess. Reserve that public
	// maximum once so AddPath can reslice the existing backing store instead of
	// allocating a new heap buffer on every ORAM access.
	stashBackingCapacity := stashSize
	var zero T
	if _, fixedValue := any(zero).(FixedValue); fixedValue {
		maxInt := int(^uint(0) >> 1)
		pathLength := bits.Len64(server.LeafCount())
		if pathLength > 0 && bucketSize > maxInt/pathLength {
			return nil, fmt.Errorf("eoram: path backing capacity overflow")
		}
		pathBlockCapacity := pathLength * bucketSize
		if stashSize > maxInt-pathBlockCapacity {
			return nil, fmt.Errorf("eoram: stash backing capacity overflow")
		}
		stashBackingCapacity += pathBlockCapacity
	}

	client := &Client[T]{
		server: server,
		stash: Stash[NodeBlock[T]]{
			blocks:        make([]StashedBlock[NodeBlock[T]], stashSize, stashBackingCapacity),
			capacity:      stashSize,
			fixedCapacity: true,
		},
		random:        cryptoRand.Reader,
		bucketSize:    bucketSize,
		nextFreeBlock: 1,
		access: accessState{
			phase: accessReady,
		},
	}
	client.pathScratch = makeBucketPath[NodeBlock[T]](
		bits.Len64(server.LeafCount()),
		bucketSize,
	)
	if err := client.initializeTree(); err != nil {
		return nil, err
	}

	return client, nil
}

// ランダムにリーフを選定する
func (client *Client[T]) NewPosition() (avl.ORAMPosition, error) {
	client.stats.NewPositions++
	if client.profileSequentialPositions {
		client.profileNextPosition = (client.profileNextPosition + 1) &
			(client.server.LeafCount() - 1)
		return avl.ORAMPosition(client.profileNextPosition), nil
	}
	if _, err := io.ReadFull(client.random, client.randomBytes[:]); err != nil {
		return 0, err
	}

	randomValue := binary.LittleEndian.Uint64(client.randomBytes[:])
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
	path, err := readBucketPathInto(&client.server, oldLeaf, client.pathScratch)
	if err != nil {
		return zeroNode, zeroToken, err
	}

	added := client.stash.AddPath(path, client.bucketSize)
	client.stats.PathBucketsRead += uint64(len(path))
	client.stats.AddPathBlocks += uint64(len(path) * client.bucketSize)
	validStashBlocks := client.stash.validBlockCount()
	if uint64(validStashBlocks) > client.stats.MaxStashLength {
		client.stats.MaxStashLength = uint64(validStashBlocks)
	}
	if added == 0 {
		return zeroNode, zeroToken, ErrStashOverflow
	}

	blockAddress := BlockAddress(pointer.Address)
	prepareStashLength := len(client.stash.blocks)
	value, found := client.stash.PrepareAccess(
		blockAddress,
		Leaf(newLeaf),
	)
	client.stats.PrepareStashScans += uint64(prepareStashLength)

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
	updated := client.stash.UpdateAccessedBlock(
		client.access.blockAddress,
		NodeBlock[T]{Node: node},
	)
	client.stats.UpdateStashScans += uint64(updateStashLength)
	if updated == 0 {
		return fmt.Errorf(
			"%w: FinishAccess blockAddress=%d",
			ErrBlockNotFound,
			client.access.blockAddress,
		)
	}

	var path []Bucket[NodeBlock[T]]
	if client.nonObliviousEviction {
		var visits uint64
		path, visits = client.stash.EvictToPathNonObliviousInto(
			client.pathScratch,
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
		path = client.stash.EvictToPathInto(
			client.pathScratch,
			client.access.oldLeaf,
			client.access.pathLength,
			client.bucketSize,
		)
	}
	client.stats.UnpinStashScans += uint64(len(client.stash.blocks))
	if err := writeBucketPathFromScratch(&client.server, path, client.access.oldLeaf); err != nil {
		return err
	}

	unpinned := client.stash.Unpin(client.access.blockAddress)
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

func makeBucketPath[T core.Selectable[T]](pathLength int, bucketSize int) []Bucket[T] {
	path := make([]Bucket[T], pathLength)
	blocks := make([]Block[T], pathLength*bucketSize)
	for depth := 0; depth < pathLength; depth++ {
		start := depth * bucketSize
		path[depth] = blocks[start : start+bucketSize]
	}
	return path
}

func (client *Client[T]) Stats() ClientStats { return client.stats }

// EnableNonObliviousEvictionForProfile enables the branchful C++ comparison
// mode. It is intentionally explicit because this weakens strong obliviousness.
func (client *Client[T]) EnableNonObliviousEvictionForProfile() {
	client.nonObliviousEviction = true
}

// EnableSequentialPositionsForProfile replaces random leaf assignment with the
// public artifact's incrementing-position shortcut. This is only for diagnosing
// the cost of crypto/rand and MUST NOT be used for security measurements.
func (client *Client[T]) EnableSequentialPositionsForProfile() {
	client.profileSequentialPositions = true
	client.profileNextPosition = 0
}

// EnableVariableStashForProfile switches the fixed-capacity stash to the
// variable-length behavior used by the public EnigMap artifact. This is only
// for diagnosing sort/work-set cost and MUST NOT be used for strong or doubly
// oblivious security measurements.
func (client *Client[T]) EnableVariableStashForProfile() {
	client.stash.enableVariableLengthForProfile()
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
