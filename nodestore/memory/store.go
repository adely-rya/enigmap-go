package memory

import (
	"errors"

	"github.com/adely-rya/enigmap-go/avl"
)

var ErrNodeNotFound = errors.New("node not found")
var ErrAccessInProgress = errors.New("node access already in progress")
var ErrNoAccessInProgress = errors.New("no node access in progress")
var ErrAccessIDMismatch = errors.New("node access ID mismatch")

type Store[V any] struct {
	nextID avl.NodeID
	nodes  map[avl.NodeID]avl.Node[V]
	active avl.NodeID
}

func New[V any]() *Store[V] {
	return &Store[V]{
		nextID: 1,
		nodes:  make(map[avl.NodeID]avl.Node[V]),
	}
}

func (s *Store[V]) Allocate(node avl.Node[V]) (avl.Pointer, error) {
	id := s.nextID
	s.nextID++
	s.nodes[id] = node
	return avl.Pointer{Address: id}, nil
}

func (s *Store[V]) NewPosition() avl.Position {
	return 0
}

func (s *Store[V]) BeginAccess(
	pointer avl.Pointer,
	newPosition avl.Position,
) (avl.Node[V], error) {
	if s.active != avl.NullAddress {
		return avl.Node[V]{}, ErrAccessInProgress
	}
	node, ok := s.nodes[pointer.Address]
	if !ok {
		return avl.Node[V]{}, ErrNodeNotFound
	}
	s.active = pointer.Address
	return node, nil
}

func (s *Store[V]) FinishAccess(id avl.NodeID, node avl.Node[V]) error {
	if s.active == avl.NullAddress {
		return ErrNoAccessInProgress
	}
	if s.active != id {
		return ErrAccessIDMismatch
	}
	if _, ok := s.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	s.nodes[id] = node
	s.active = avl.NullAddress
	return nil
}

// FreeMaybe frees id when condition is 1. The memory backend is intended for
// functional testing; an ORAM backend must keep the physical access trace
// independent of condition.
func (s *Store[V]) FreeMaybe(pointer avl.Pointer, condition int) error {
	if condition&1 == 0 {
		return nil
	}
	if _, ok := s.nodes[pointer.Address]; !ok {
		return ErrNodeNotFound
	}
	delete(s.nodes, pointer.Address)
	return nil
}
