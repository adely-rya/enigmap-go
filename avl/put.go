package avl

func (t *Tree[V]) Put(key []byte, value V) error {
	candidate := Node[V]{
		Key:     NewKey(key),
		Value:   value,
		Balance: BalanceBalanced,
		Left:    t.fake,
		Right:   t.fake,
		Valid:   1,
	}

	_, err := t.store.Allocate(candidate)
	return err
}
