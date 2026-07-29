package avl

import "crypto/subtle"

func (t *Tree[V]) Put(key []byte, value V) error {
	insertKey := NewKey(key)

	candidate := Node[V]{
		Key:     insertKey,
		Value:   value,
		Balance: BalanceBalanced,
		Left:    t.fake,
		Right:   t.fake,
		Valid:   1,
	}

	//いったん新たなノードを挿入しておく
	//もしkeyがすでに存在していたら後でこのAllocateしたノードは消す　ifを消すために必ずAllocate
	candidateID, err := t.store.Allocate(candidate)

	currentID := t.root
	keyExists := 0           //引数と同じkeyが存在していたか否か判定
	insertionPointFound := 0 //挿入ポイント判定
	var existingNodeID NodeID
	var insertionParentID NodeID
	var insertionRight int

	for depth := 0; depth < t.maxDepth; depth++ {
		node, err := t.store.Load(currentID)
		if err != nil {
			return err
		}

		comparison := CompareKey(insertKey, node.Key)
		equal := subtle.ConstantTimeEq(int32(comparison), 0) // if key == node.Key { 1 } else { 0 }; 一致なら1
		right := subtle.ConstantTimeEq(int32(comparison), 1) // if key > node.Key { 1 } else { 0 }; 右へ進むなら1
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)    // if node is real { 1 } else { 0 }; 実ノードなら1

		match := equal & valid // if equal == 1 && valid == 1 { 1 } else { 0 }; 実ノードで一致なら1
		right &= valid         // if valid == 0 { right = 0 }; dummyは左へ

		existingNodeID = selectNodeID(existingNodeID, currentID, match)
		keyExists |= match // if match == 1 { keyExists = 1 }; 一度一致したら1

		next := selectNodeID(node.Left, node.Right, right)

		nextIsFake := nodeIDPresent(next^t.fake) ^ 1         // if next == fake { 1 } else { 0 }; 次がfakeなら1
		currentIsFake := nodeIDPresent(currentID^t.fake) ^ 1 // if current == fake { 1 } else { 0 }; 現在がfakeなら1
		foundInsertionPoint := nextIsFake &
			(currentIsFake ^ 1) &
			(keyExists ^ 1) &
			(insertionPointFound ^ 1) // if first missing child { 1 } else { 0 }; 最初の空きなら1

		insertionParentID = selectNodeID(
			insertionParentID,
			currentID,
			foundInsertionPoint,
		)
		insertionRight = subtle.ConstantTimeSelect(
			foundInsertionPoint,
			right,
			insertionRight,
		)
		insertionPointFound |= foundInsertionPoint // if found { insertionPointFound = 1 }; 発見後は1

		currentID = selectNodeID(t.fake, next, nodeIDPresent(next))
	}

	err = t.store.FreeMaybe(candidateID, keyExists)
	if err != nil {
		return err
	}

	for depth := 0; depth < t.maxDepth; depth++ {

	}

	return err
}
