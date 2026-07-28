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

func (s *Store[V]) Allocate(node avl.Node[V]) (avl.NodeID, error) {
	id := s.nextID
	s.nextID++
	s.nodes[id] = node
	return id, nil
}

func (s *Store[V]) Load(id avl.NodeID) (avl.Node[V], error) {
	node, ok := s.nodes[id]
	if !ok {
		return avl.Node[V]{}, ErrNodeNotFound
	}
	return node, nil
}

func (s *Store[V]) Update(id avl.NodeID, node avl.Node[V]) error {
	if _, ok := s.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	s.nodes[id] = node
	return nil
}

// FreeMaybe frees id when condition is 1. The memory backend is intended for
// functional testing; an ORAM backend must keep the physical access trace
// independent of condition.
func (s *Store[V]) FreeMaybe(id avl.NodeID, condition int) error {
	if condition&1 == 0 {
		return nil
	}
	if _, ok := s.nodes[id]; !ok {
		return ErrNodeNotFound
	}
	delete(s.nodes, id)
	return nil
}
