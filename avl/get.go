package avl

import "crypto/subtle"

func (t *Tree[V]) Get(key []byte) (V, bool, error) {
	searchKey := NewKey(key)

	current := t.root
	currentNewPosition, err := t.store.NewPosition()
	if err != nil {
		var zero V
		return zero, false, err
	}
	t.root.Position = currentNewPosition
	found := 0
	var result V

	for depth := 0; depth < t.maxDepth; depth++ {
		nextNewPosition, err := t.store.NewPosition()
		if err != nil {
			var zero V
			return zero, false, err
		}

		node, token, err := t.store.BeginAccess(current, currentNewPosition)
		if err != nil {
			var zero V
			return zero, false, err
		}
		currentNew := pointerWithPosition(current, currentNewPosition)
		t.fake = selectPointer(
			t.fake,
			currentNew,
			pointerAddressesEqual(current, t.fake),
		)

		comparison := CompareKey(searchKey, node.Key)
		equal := subtle.ConstantTimeEq(int32(comparison), 0)
		right := subtle.ConstantTimeEq(int32(comparison), 1)
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)
		match := equal & valid
		right &= valid

		result = selectValue(result, node.Value, match)
		found |= match

		next := selectPointer(node.Left, node.Right, right)
		nextNew := pointerWithPosition(next, nextNewPosition)
		node.Left = selectPointer(node.Left, nextNew, conditionNot(right))
		node.Right = selectPointer(node.Right, nextNew, right)

		if err := t.store.FinishAccess(token, node); err != nil {
			var zero V
			return zero, false, err
		}
		next = selectPointer(
			next,
			t.fake,
			pointerAddressesEqual(next, t.fake),
		)

		current = selectPointer(
			t.fake,
			next,
			isNotNullPointer(next),
		)
		currentNewPosition = nextNewPosition
	}

	return result, found == 1, nil
}
