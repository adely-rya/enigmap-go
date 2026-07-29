package avl

import "crypto/subtle"

type rotationType uint8

const (
	rotationNone      rotationType = iota //新しい傾きだけを更新
	rotationDirect                        //balancedに戻して回転不要
	rotationTwoNode                       //単回転
	rotationThreeNode                     //二重回転
)

type putMetadata struct {
	// 新しいノードの接続先
	insertionParentID NodeID //親のNodeID
	insertionRight    int    //0ならLeft、1ならRightに挿入

	// 既存キーの更新先
	keyExists      int
	existingNodeID NodeID

	// balance更新・回転の基準
	pivotID        NodeID
	pivotBalance   Balance
	pivotParentID  NodeID
	pivotChildID   NodeID
	pivotDirection int
	childDirection int
	rotation       rotationType
}

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

	metadata := putMetadata{
		rotation: rotationNone,
	}

	currentID := t.root
	parentID := NullAddress
	insertionPointFound := 0 //挿入ポイント判定

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

		metadata.existingNodeID = selectNodeID(
			metadata.existingNodeID,
			currentID,
			match,
		)
		metadata.keyExists |= match // if match == 1 { keyExists = 1 }; 一度一致したら1

		next := selectNodeID(node.Left, node.Right, right)

		pivotExists := nodeIDPresent(metadata.pivotID)
		currentIsPivotChild := (nodeIDPresent(currentID^metadata.pivotChildID) ^ 1) & pivotExists // if current == pivotChild && pivot exists { 1 } else { 0 }; pivotの子なら1
		metadata.childDirection = selectInt(
			metadata.childDirection,
			right,
			currentIsPivotChild,
		)

		balanced := subtle.ConstantTimeByteEq(
			byte(node.Balance),
			byte(BalanceBalanced),
		) //balanced(左右の高さが同じ)なら1 非balancedなら0
		pivotCandidate := valid & (balanced ^ 1) // if real && not balanced { 1 } else { 0 }; 非balancedなら1
		metadata.pivotID = selectNodeID(
			metadata.pivotID,
			currentID,
			pivotCandidate,
		) //node.Valid == 1 && node.Balance != BalanceBalanced なら回転の可能性があるので今のブロックを保存
		metadata.pivotParentID = selectNodeID(
			metadata.pivotParentID,
			parentID,
			pivotCandidate,
		) //node.Valid == 1 && node.Balance != BalanceBalanced なら回転の可能性があるので今の親ブロックを保存
		metadata.pivotChildID = selectNodeID(
			metadata.pivotChildID,
			next,
			pivotCandidate,
		) //node.Valid == 1 && node.Balance != BalanceBalanced なら回転の可能性があるので今のブロックの子を保存
		metadata.pivotDirection = selectInt(
			metadata.pivotDirection,
			right,
			pivotCandidate,
		) //node.Valid == 1 && node.Balance != BalanceBalanced なら回転の可能性があるので今のブロックを保存
		metadata.pivotBalance = selectBalance(
			metadata.pivotBalance,
			node.Balance,
			pivotCandidate,
		) //node.Valid == 1 && node.Balance != BalanceBalanced なら元のbalanceを保存

		nextIsFake := nodeIDPresent(next^t.fake) ^ 1         // if next == fake { 1 } else { 0 }; 次がfakeなら1
		currentIsFake := nodeIDPresent(currentID^t.fake) ^ 1 // if current == fake { 1 } else { 0 }; 現在がfakeなら1
		foundInsertionPoint := nextIsFake &
			(currentIsFake ^ 1) &
			(metadata.keyExists ^ 1) &
			(insertionPointFound ^ 1) // if first missing child { 1 } else { 0 }; 最初の空きなら1

		metadata.insertionParentID = selectNodeID( //挿入する場所の親を保存
			metadata.insertionParentID,
			currentID,
			foundInsertionPoint,
		)
		metadata.insertionRight = selectInt( //親から見てRightとLeftのどちらに挿入するかを保存
			metadata.insertionRight,
			right,
			foundInsertionPoint,
		)
		insertionPointFound |= foundInsertionPoint // if found { insertionPointFound = 1 }; 発見後は1

		parentID = selectNodeID(parentID, currentID, currentIsFake^1) // if current != fake { parent = current }; 最後の親を保持
		currentID = selectNodeID(t.fake, next, nodeIDPresent(next))
	}

	//操作種別の判定
	pivotExists := nodeIDPresent(metadata.pivotID)
	shouldRebalance := pivotExists &
		(metadata.keyExists ^ 1) &
		insertionPointFound // if pivot exists && key is new && insertion point found { 1 }; 操作判定が必要なのか？（例えばもしKeyがすでにあるなら操作いらない）
	pivotFollowsBalance := subtle.ConstantTimeByteEq(
		byte(metadata.pivotBalance),
		byte(metadata.pivotDirection),
	) // if pivotBalance == pivotDirection { 1 } else { 0 }; 高い側へ挿入しているか挿入されていないかを判定　バランスが悪化するかしないか
	childFollowsPivot := subtle.ConstantTimeEq(
		int32(metadata.childDirection),
		int32(metadata.pivotDirection),
	) // if childDirection == pivotDirection { 1 } else { 0 }; RR LL → 1  RL LR → 0

	directRebalance := shouldRebalance &
		(pivotFollowsBalance ^ 1) // if opposite direction { 1 } else { 0 }; balancedへ戻るなら1
	twoNodeRotation := shouldRebalance &
		pivotFollowsBalance &
		childFollowsPivot // if LL or RR { 1 } else { 0 }; 単回転なら1
	threeNodeRotation := shouldRebalance &
		pivotFollowsBalance &
		(childFollowsPivot ^ 1) // if LR or RL { 1 } else { 0 }; 二重回転なら1

	/*
	  A = 1：新規挿入で、pivotも挿入位置も存在する
	  B = 1：pivotが高い方向へ挿入した
	  C = 1：pivotとpivot childで同じ方向へ進んだ

	  コードは次の論理式になる。

	  回転不要 = A AND NOT B
	  二重回転   = A AND B AND C
	  三重回転  = A AND B AND NOT C
	*/

	metadata.rotation = selectRotationType(
		metadata.rotation,
		rotationDirect,
		directRebalance,
	)
	metadata.rotation = selectRotationType(
		metadata.rotation,
		rotationTwoNode,
		twoNodeRotation,
	)
	metadata.rotation = selectRotationType(
		metadata.rotation,
		rotationThreeNode,
		threeNodeRotation,
	)

	err = t.store.FreeMaybe(candidateID, metadata.keyExists)
	if err != nil {
		return err
	}

	for depth := 0; depth < t.maxDepth; depth++ {

	}

	return err
}
