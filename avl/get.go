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
		equal := subtle.ConstantTimeEq(int32(comparison), 0) //keyのマッチ判定 イコールなら1
		right := subtle.ConstantTimeEq(int32(comparison), 1) //rightに進むか判定  rightなら1
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)    //ダミーか判定　実nodeなら1
		match := equal & valid                               //ダミーじゃない、かつkeyが一致したか判定

		result = t.selector.Select(result, node.Value, match)
		found |= match //すでに見つかったのかを判定　探しているkeyが見つかっていたらfound = 1

		next := selectNodeID(node.Left, node.Right, right)
		current = selectNodeID(
			t.fake,
			next,
			nodeIDPresent(next),
		)
	}

	return result, found == 1, nil
}
