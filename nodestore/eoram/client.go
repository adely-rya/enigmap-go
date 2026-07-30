// Package eoram stores AVL nodes in ENIGMAP's non-recursive Path ORAM.
package eoram

import (
	"errors"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
	"github.com/adely-rya/oram/core"
)

var ErrNotImplemented = errors.New("eoram: not implemented")

// ORAMClient is the AVL node store backed by Path ORAM.
//
// Unlike the generic Path ORAM client, it does not have a central position
// map. AVL child pointers will carry their logical address and ORAM position.
type ORAMClient struct {
	Stash      Stash
	Server     ORAMServer[Bucket, core.IO[Bucket]]
	Capacity   uint64
	BucketSize int

	access AccessState
}

// Allocate reserves a logical address and position and stores a new AVL node.
func (client *ORAMClient) Allocate(
	node avl.Node[omap.Value],
) (avl.Pointer, error) {
	return avl.NullPointer, ErrNotImplemented
}

func (client *ORAMClient) NewPosition() avl.Position {
	panic(ErrNotImplemented)
}

// BeginAccess opens one node access. Only one access may be open at a time.
// pointer contains the old path and newPosition is assigned to the block.
func (client *ORAMClient) BeginAccess(
	pointer avl.Pointer,
	newPosition avl.Position,
) (avl.Node[omap.Value], error) {
	return avl.Node[omap.Value]{}, ErrNotImplemented
}

// FinishAccess updates the block opened by BeginAccess and evicts its saved
// path without reading that path a second time.
func (client *ORAMClient) FinishAccess(
	id avl.NodeID,
	node avl.Node[omap.Value],
) error {
	return ErrNotImplemented
}

// FreeMaybe releases the most recently allocated candidate when condition is
// one while keeping the physical ORAM access schedule independent of it.
func (client *ORAMClient) FreeMaybe(
	pointer avl.Pointer,
	condition int,
) error {
	return ErrNotImplemented
}

// UnstickAll makes every stashed block eligible for later eviction.
func (client *ORAMClient) UnstickAll() {
	// TODO: clear the Sticky flag with a fixed stash scan.
}

var _ avl.NodeStore[omap.Value] = (*ORAMClient)(nil)
