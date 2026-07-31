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
		equal := subtle.ConstantTimeEq(int32(comparison), 0) // if key == node.Key { 1 } else { 0 }; 一致なら1
		right := subtle.ConstantTimeEq(int32(comparison), 1) // if key > node.Key { 1 } else { 0 }; 右へ進むなら1
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)    // if node is real { 1 } else { 0 }; 実ノードなら1
		match := equal & valid                               // if equal == 1 && valid == 1 { 1 } else { 0 }; 実ノードで一致なら1
		right &= valid                                       // if valid == 0 { right = 0 }; dummyは左へ

		result = selectValue(result, node.Value, match)
		found |= match // if match == 1 { found = 1 }; 一度一致したら1

		next := selectPointer(node.Left, node.Right, right)
		nextNew := pointerWithPosition(next, nextNewPosition)
		node.Left = selectPointer(node.Left, nextNew, conditionNot(right))
		node.Right = selectPointer(node.Right, nextNew, right)

		if err := t.store.FinishAccess(token, node); err != nil {
			var zero V
			return zero, false, err
		}

		current = selectPointer(
			t.fake,
			next,
			isNotNullPointer(next),
		)
		currentNewPosition = nextNewPosition
	}

	return result, found == 1, nil
}
