package memory

import (
	"errors"

	"github.com/adely-rya/enigmap-go/avl"
)

var ErrNodeNotFound = errors.New("node not found")

type Store[V any] struct {
	nextID avl.NodeID
	nodes  map[avl.NodeID]avl.Node[V]
}

func New[V any]() *Store[V] {
	return &Store[V]{
		nextID: 1,
		nodes:  make(map[avl.NodeID]avl.Node[V]),
	}
}

func (s *Store[V]) NewPosition() (avl.ORAMPosition, error) {
	return 0, nil
}

func (s *Store[V]) Allocate(node avl.Node[V]) (avl.Pointer, error) {
	id := s.nextID
	s.nextID++
	s.nodes[id] = node
	return avl.Pointer{Address: id}, nil
}

func (s *Store[V]) BeginAccess(
	pointer avl.Pointer,
	_ avl.ORAMPosition,
) (avl.Node[V], avl.AccessToken, error) {
	node, ok := s.nodes[pointer.Address]
	if !ok {
		return avl.Node[V]{}, avl.AccessToken{}, ErrNodeNotFound
	}
	return node, avl.AccessToken{Address: pointer.Address}, nil
}

func (s *Store[V]) FinishAccess(token avl.AccessToken, node avl.Node[V]) error {
	if _, ok := s.nodes[token.Address]; !ok {
		return ErrNodeNotFound
	}
	s.nodes[token.Address] = node
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

func (s *Store[V]) UncacheAll() {}
