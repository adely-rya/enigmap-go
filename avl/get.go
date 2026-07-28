package avl

import "crypto/subtle"

func (t *Tree[V]) Get(key []byte) (V, bool, error) {
	searchKey := NewKey(key)

	current := selectNodeID(
		t.fake,
		t.root,
		nodeIDPresent(t.root),
	)
	found := 0
	var result V

	for depth := 0; depth < t.maxDepth; depth++ {
		node, err := t.store.Load(current)
		if err != nil {
			var zero V
			return zero, false, err
		}

		comparison := CompareKey(searchKey, node.Key)
		equal := subtle.ConstantTimeEq(int32(comparison), 0)
		right := subtle.ConstantTimeEq(int32(comparison), 1)
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)
		match := equal & valid

		result = t.selector.Select(result, node.Value, match)
		found |= match

		next := selectNodeID(node.Left, node.Right, right)
		current = selectNodeID(
			t.fake,
			next,
			nodeIDPresent(next),
		)
	}

	return result, found == 1, nil
}
