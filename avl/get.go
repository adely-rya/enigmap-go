package avl

func (t *Tree[V]) Get(key []byte) (V, bool, error) {
	searchKey := NewKey(key)

	id := t.root

	for id != NullAddress {
		node, err := t.store.Load(id)
		if err != nil {
			var zero V
			return zero, false, err
		}

		switch CompareKey(searchKey, node.Key) {
		case 0:
			return node.Value, true, nil
		case -1:
			id = node.Left
		default:
			id = node.Right
		}
	}

	var zero V
	return zero, false, nil
}
