package avl

import "crypto/subtle"

func (t *Tree[V]) Get(key []byte) (V, bool, error) {
	searchKey := NewKey(key)

	current := t.root
	newPosition := t.store.NewPosition()
	rootPosition := newPosition
	found := 0
	var result V

	for depth := 0; depth < t.maxDepth; depth++ {
		currentPointer := current
		node, err := t.store.BeginAccess(current, newPosition)
		if err != nil {
			var zero V
			return zero, false, err
		}
		current.Position = newPosition
		currentIsFake := pointerAddressesEqual(current, t.fake)
		t.fake.Position = selectPosition(t.fake.Position, newPosition, currentIsFake)

		comparison := CompareKey(searchKey, node.Key)
		equal := subtle.ConstantTimeEq(int32(comparison), 0) // if key == node.Key { 1 } else { 0 }; 一致なら1
		right := subtle.ConstantTimeEq(int32(comparison), 1) // if key > node.Key { 1 } else { 0 }; 右へ進むなら1
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)    // if node is real { 1 } else { 0 }; 実ノードなら1
		match := equal & valid                               // if equal == 1 && valid == 1 { 1 } else { 0 }; 実ノードで一致なら1
		right &= valid                                       // if valid == 0 { right = 0 }; dummyは左へ

		result = selectValue(result, node.Value, match)
		found |= match // if match == 1 { found = 1 }; 一度一致したら1

		next := selectPointer(node.Left, node.Right, right)
		newPosition = t.store.NewPosition()
		updatedNext := next
		updatedNext.Position = newPosition
		node.Left = selectPointer(node.Left, updatedNext, conditionNot(right))
		node.Right = selectPointer(node.Right, updatedNext, right)
		current = selectPointer(
			t.fake,
			next,
			isNotNullAddress(next.Address),
		)

		if err := t.store.FinishAccess(currentPointer.Address, node); err != nil {
			var zero V
			return zero, false, err
		}
	}
	t.root.Position = rootPosition

	return result, found == 1, nil
}
