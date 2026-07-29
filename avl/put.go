package avl

import (
	"crypto/subtle"
)

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
	pivotID              NodeID
	pivotBalance         Balance
	pivotParentID        NodeID
	pivotChildID         NodeID
	pivotChildOppositeID NodeID
	pivotGrandchildID    NodeID
	grandchildLeftID     NodeID
	grandchildRightID    NodeID
	pivotDirection       int
	childDirection       int
	grandchildDirection  int
	rotation             rotationType
}

// conditionNot は、0/1で表した条件を反転する。
// conditionが0なら1、1なら0を返す。
// secretに応じた分岐を作らないため、ifではなくビット演算で反転する。
func conditionNot(condition int) int {
	return condition ^ 1
}

// conditionsBoth は、2つの0/1条件が両方とも成立するときだけ1を返す。
// AND条件の計算をPut本体へ直接書かず、判定の意図を読みやすくするために使う。
func conditionsBoth(first, second int) int {
	return first & second
}

// conditionsAll3 は、3つの0/1条件がすべて成立するときだけ1を返す。
func conditionsAll3(first, second, third int) int {
	return first & second & third
}

// conditionsAll4 は、4つの0/1条件がすべて成立するときだけ1を返す。
func conditionsAll4(first, second, third, fourth int) int {
	return first & second & third & fourth
}

// conditionsAny3 は、3つの0/1条件のうち1つでも成立すると1を返す。
func conditionsAny3(first, second, third int) int {
	return first | second | third
}

// rememberCondition は、過去または今回のどちらかで条件が成立していれば1を返す。
// key発見済み・挿入位置発見済みという状態を、その後も1のまま保持するために使う。
func rememberCondition(previous, current int) int {
	return previous | current
}

// matchingRealNode は、現在のノードが実ノードで、かつkeyも一致するときに1を返す。
// fakeはkey領域が偶然一致しても検索結果にしてはいけないため、validも必須とする。
func matchingRealNode(keysEqual, valid int) int {
	return conditionsBoth(keysEqual, valid)
}

// directionForRealNode は、実ノードなら比較結果の探索方向を返し、fakeなら0を返す。
// 返り値は0がLeft、1がRight。fake以降のdummy走査は常にLeftへ揃える。
func directionForRealNode(right, valid int) int {
	return conditionsBoth(right, valid)
}

// currentIsStoredNode は、現在IDが保存済みIDと一致し、そのIDが存在するときに1を返す。
// NullAddress同士の一致を、pivot・child・grandchildの一致と誤認しないために使う。
func currentIsStoredNode(currentID, storedID NodeID, storedNodeExists int) int {
	return conditionsBoth(nodeIDsEqual(currentID, storedID), storedNodeExists)
}

// isPivotCandidate は、現在のノードが実ノードかつ非Balancedなら1を返す。
// 挿入経路上で最後に見つかった非Balancedノードをpivot候補として保存する。
func isPivotCandidate(valid, balanced int) int {
	return conditionsBoth(valid, conditionNot(balanced))
}

// isFirstInsertionPoint は、今回初めて「実ノードの次がfake」に到達したときに1を返す。
// 既存keyの更新時、fake上のdummy走査、挿入位置を保存済みの場合は0を返す。
func isFirstInsertionPoint(nextIsFake, currentIsFake, keyExists, insertionPointFound int) int {
	return conditionsAll4(
		nextIsFake,
		conditionNot(currentIsFake),
		conditionNot(keyExists),
		conditionNot(insertionPointFound),
	)
}

// candidateBecomesPivotGrandchild は、candidateをpivot childの直下へ挿入するときに1を返す。
// この場合、走査前に計算したnextはfakeなので、孫IDをcandidateIDへ補正する必要がある。
func candidateBecomesPivotGrandchild(currentIsPivotChild, foundInsertionPoint int) int {
	return conditionsBoth(currentIsPivotChild, foundInsertionPoint)
}

// shouldInsertLeft は、見つけた挿入位置が親のLeft側なら1を返す。
func shouldInsertLeft(foundInsertionPoint, right int) int {
	return conditionsBoth(foundInsertionPoint, conditionNot(right))
}

// shouldInsertRight は、見つけた挿入位置が親のRight側なら1を返す。
func shouldInsertRight(foundInsertionPoint, right int) int {
	return conditionsBoth(foundInsertionPoint, right)
}

// shouldClassifyRebalance は、新規挿入が完了し、pivotも存在するときに1を返す。
// 既存値の上書きや、挿入位置が見つからなかった場合はbalanceも回転も変更しない。
func shouldClassifyRebalance(pivotExists, keyExists, insertionPointFound int) int {
	return conditionsAll3(pivotExists, conditionNot(keyExists), insertionPointFound)
}

// isDirectRebalance は、pivotの低い側へ挿入してBalancedへ戻る操作なら1を返す。
// 木の高さは変わらないため回転は不要。
func isDirectRebalance(shouldRebalance, pivotFollowsBalance int) int {
	return conditionsBoth(shouldRebalance, conditionNot(pivotFollowsBalance))
}

// isTwoNodeRotation は、pivotの高い側へ同じ方向に2回進むLL/RRなら1を返す。
// LLは右単回転、RRは左単回転を行う。
func isTwoNodeRotation(shouldRebalance, pivotFollowsBalance, childFollowsPivot int) int {
	return conditionsAll3(shouldRebalance, pivotFollowsBalance, childFollowsPivot)
}

// isThreeNodeRotation は、pivotの高い側へ進んだ後に逆方向へ進むLR/RLなら1を返す。
// LR/RLではpivot grandchildを新しい部分木のrootにする二重回転を行う。
func isThreeNodeRotation(shouldRebalance, pivotFollowsBalance, childFollowsPivot int) int {
	return conditionsAll3(
		shouldRebalance,
		pivotFollowsBalance,
		conditionNot(childFollowsPivot),
	)
}

// isNewInsertion は、既存keyの更新ではなく、新しいcandidateを接続できたときに1を返す。
func isNewInsertion(keyExists, insertionPointFound int) int {
	return conditionsBoth(conditionNot(keyExists), insertionPointFound)
}

// shouldStartPathBalance は、新規挿入で、かつpivotが存在しないrotationNoneなら1を返す。
// rotationNoneではrootからcandidateの親まで、挿入方向に沿ってbalanceを更新する。
func shouldStartPathBalance(newInsertion, rotationIsNone int) int {
	return conditionsBoth(newInsertion, rotationIsNone)
}

// operationAtNode は、指定した操作が選ばれ、かつ現在その対象ノードにいるときに1を返す。
// 例: operationAtNode(twoNodeRotation, atPivot) は「単回転のpivot上」を表す。
func operationAtNode(operationSelected, atNode int) int {
	return conditionsBoth(operationSelected, atNode)
}

// shouldUpdatePathBalance は、balance更新区間内の実ノードでcandidate以外なら1を返す。
// candidate自身は常にBalancedで生成されるため、挿入方向への傾きで上書きしない。
func shouldUpdatePathBalance(rebalanceActive, valid, atCandidate int) int {
	return conditionsAll3(rebalanceActive, valid, conditionNot(atCandidate))
}

// shouldReplaceParentLink は、回転対象の親にいて、指定した子がpivotなら1を返す。
// 親のLeft/Rightのうち、実際にpivotを指していた側だけを新しい部分木rootへ付け替える。
func shouldReplaceParentLink(atRotationParent, childIsPivot int) int {
	return conditionsBoth(atRotationParent, childIsPivot)
}

// shouldApplyDirection は、対象ノード上にいて、指定方向の更新を行うときに1を返す。
// pointer更新をLeft用とRight用の両方へ記述しつつ、必要な片側だけを選択する。
func shouldApplyDirection(atRotationNode, direction int) int {
	return conditionsBoth(atRotationNode, direction)
}

// isInsertedGrandchild は、二重回転のpivot grandchildが今回のcandidate自身なら1を返す。
// candidate自身なら左右に既存部分木がなく、pivotとpivot childは両方Balancedになる。
func isInsertedGrandchild(pivotGrandchildID, candidateID NodeID, newInsertion int) int {
	return conditionsBoth(nodeIDsEqual(pivotGrandchildID, candidateID), newInsertion)
}

// shouldTiltPivotOpposite は、二重回転後のpivotを挿入方向と逆へ傾ける場合に1を返す。
// grandchildがcandidate自身ではなく、grandchildからpivotと同方向へ進んだLR/RLが対象。
func shouldTiltPivotOpposite(
	threeAtPivot,
	grandchildIsCandidate,
	grandchildFollowsPivot int,
) int {
	return conditionsAll3(
		threeAtPivot,
		conditionNot(grandchildIsCandidate),
		grandchildFollowsPivot,
	)
}

// shouldTiltChildTowardPivot は、二重回転後のpivot childをpivot方向へ傾ける場合に1を返す。
// grandchildがcandidate自身ではなく、grandchildからpivotと逆方向へ進んだLR/RLが対象。
func shouldTiltChildTowardPivot(
	threeAtPivotChild,
	grandchildIsCandidate,
	grandchildFollowsPivot int,
) int {
	return conditionsAll3(
		threeAtPivotChild,
		conditionNot(grandchildIsCandidate),
		conditionNot(grandchildFollowsPivot),
	)
}

// isThreeNodeRotationMember は、現在位置が二重回転を構成する3ノードのどれかなら1を返す。
// pivot・pivot child・pivot grandchildを、まず共通のBalancedへ初期化するために使う。
func isThreeNodeRotationMember(threeAtPivot, threeAtPivotChild, threeAtPivotGrandchild int) int {
	return conditionsAny3(threeAtPivot, threeAtPivotChild, threeAtPivotGrandchild)
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
	if err != nil {
		return err
	}

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

		match := matchingRealNode(equal, valid)    // 実ノードでkeyが一致するなら1
		right = directionForRealNode(right, valid) // fakeならLeft、実ノードなら比較結果の方向

		metadata.existingNodeID = selectNodeID(
			metadata.existingNodeID,
			currentID,
			match,
		)
		metadata.keyExists = rememberCondition(metadata.keyExists, match) // 一度一致したら1を保持
		node.Value = selectValue(node.Value, value, match)                // if match == 1 { valueを上書き }; 既存キーを更新

		next := selectNodeID(node.Left, node.Right, right)

		pivotExists := isNotNullAddress(metadata.pivotID)
		currentIsPivotChild := currentIsStoredNode(
			currentID,
			metadata.pivotChildID,
			pivotExists,
		) // pivotが存在し、現在位置がそのchildなら1
		metadata.childDirection = selectInt(
			metadata.childDirection,
			right,
			currentIsPivotChild,
		)
		oppositeChild := selectNodeID(
			node.Right,
			node.Left,
			right,
		) // if Leftへ進む { Right child } else { Left child }; 挿入経路の反対側の子要素も取得（回転走査で必要）
		metadata.pivotChildOppositeID = selectNodeID(
			metadata.pivotChildOppositeID,
			oppositeChild,
			currentIsPivotChild,
		)
		metadata.pivotGrandchildID = selectNodeID(
			metadata.pivotGrandchildID,
			next,
			currentIsPivotChild,
		) //pivot childの子を二重回転のために取得

		grandchildExists := isNotNullAddress(metadata.pivotGrandchildID) //孫がいるか
		currentIsPivotGrandchild := currentIsStoredNode(
			currentID,
			metadata.pivotGrandchildID,
			grandchildExists,
		) // 保存済みのpivot grandchild上なら1
		metadata.grandchildLeftID = selectNodeID(
			metadata.grandchildLeftID,
			node.Left,
			currentIsPivotGrandchild,
		) //孫のLeftノード取得
		metadata.grandchildRightID = selectNodeID(
			metadata.grandchildRightID,
			node.Right,
			currentIsPivotGrandchild,
		) //孫のRightノード取得
		metadata.grandchildDirection = selectInt(
			metadata.grandchildDirection,
			right,
			currentIsPivotGrandchild,
		)

		balanced := subtle.ConstantTimeByteEq(
			byte(node.Balance),
			byte(BalanceBalanced),
		) //balanced(左右の高さが同じ)なら1 非balancedなら0
		pivotCandidate := isPivotCandidate(valid, balanced) // 実ノードかつ非Balancedなら1
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
		metadata.pivotGrandchildID = selectNodeID(
			metadata.pivotGrandchildID,
			NullAddress,
			pivotCandidate,
		) //新たなpivot発見時に保存していたgrandchildの情報を無効化

		nextIsFake := nodeIDsEqual(next, t.fake)         // if next == fake { 1 } else { 0 }; 次がfakeなら1
		currentIsFake := nodeIDsEqual(currentID, t.fake) // if current == fake { 1 } else { 0 }; 現在がfakeなら1
		foundInsertionPoint := isFirstInsertionPoint(
			nextIsFake,
			currentIsFake,
			metadata.keyExists,
			insertionPointFound,
		) // 既存keyではなく、最初に見つけた実ノードの空きchildなら1

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
		candidateIsPivotGrandchild := candidateBecomesPivotGrandchild(
			currentIsPivotChild,
			foundInsertionPoint,
		)
		metadata.pivotGrandchildID = selectNodeID(
			metadata.pivotGrandchildID,
			candidateID,
			candidateIsPivotGrandchild,
		)
		metadata.grandchildLeftID = selectNodeID(
			metadata.grandchildLeftID,
			t.fake,
			candidateIsPivotGrandchild,
		)
		metadata.grandchildRightID = selectNodeID(
			metadata.grandchildRightID,
			t.fake,
			candidateIsPivotGrandchild,
		)
		insertionPointFound = rememberCondition(insertionPointFound, foundInsertionPoint)

		insertLeft := shouldInsertLeft(foundInsertionPoint, right)
		insertRight := shouldInsertRight(foundInsertionPoint, right)
		node.Left = selectNodeID(node.Left, candidateID, insertLeft)
		node.Right = selectNodeID(node.Right, candidateID, insertRight) //左右両方への処理をすることで実行時間の左右差を無くしている。

		if err := t.store.Update(currentID, node); err != nil {
			return err
		}

		parentID = selectNodeID(
			parentID,
			currentID,
			conditionNot(currentIsFake),
		) // currentが実ノードなら、次の走査で使う親として保持
		currentID = selectNodeID(t.fake, next, isNotNullAddress(next))
	}

	//操作種別の判定
	pivotExists := isNotNullAddress(metadata.pivotID)
	shouldRebalance := shouldClassifyRebalance(
		pivotExists,
		metadata.keyExists,
		insertionPointFound,
	) // 新規挿入が完了し、pivotもある場合だけ操作を判定
	pivotFollowsBalance := subtle.ConstantTimeByteEq(
		byte(metadata.pivotBalance),
		byte(metadata.pivotDirection),
	) // if pivotBalance == pivotDirection { 1 } else { 0 }; 高い側へ挿入しているか挿入されていないかを判定　バランスが悪化するかしないか
	childFollowsPivot := subtle.ConstantTimeEq(
		int32(metadata.childDirection),
		int32(metadata.pivotDirection),
	) // if childDirection == pivotDirection { 1 } else { 0 }; RR LL → 1  RL LR → 0

	directRebalance := isDirectRebalance(shouldRebalance, pivotFollowsBalance)
	twoNodeRotation := isTwoNodeRotation(
		shouldRebalance,
		pivotFollowsBalance,
		childFollowsPivot,
	)
	threeNodeRotation := isThreeNodeRotation(
		shouldRebalance,
		pivotFollowsBalance,
		childFollowsPivot,
	)

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

	currentID = t.root //走査のスタートノード
	newInsertion := isNewInsertion(metadata.keyExists, insertionPointFound)
	rotationIsNone := rotationTypesEqual(metadata.rotation, rotationNone) //回転が必要か
	rebalanceActive := shouldStartPathBalance(newInsertion, rotationIsNone)

	for depth := 0; depth < t.maxDepth; depth++ {
		node, err := t.store.Load(currentID)
		if err != nil {
			return err
		}

		comparison := CompareKey(insertKey, node.Key)
		right := subtle.ConstantTimeEq(int32(comparison), 1) // if key > node.Key { 1 } else { 0 }; 右へ進むなら1
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)    // if node is real { 1 } else { 0 }; 実ノードなら1
		right = directionForRealNode(right, valid)           // fakeならLeft、実ノードなら比較結果の方向
		next := selectNodeID(node.Left, node.Right, right)   // 回転などの更新走査をする前に次のアクセス先を明らかに

		atPivot := currentIsStoredNode(currentID, metadata.pivotID, pivotExists)
		atPivotParent := currentIsStoredNode(currentID, metadata.pivotParentID, pivotExists)
		atPivotChild := currentIsStoredNode(currentID, metadata.pivotChildID, pivotExists)
		atPivotGrandchild := currentIsStoredNode(
			currentID,
			metadata.pivotGrandchildID,
			pivotExists,
		)
		atCandidate := currentIsStoredNode(currentID, candidateID, newInsertion)

		directRotation := rotationTypesEqual(
			metadata.rotation,
			rotationDirect,
		) //回転不要で挿入したらバランスが取れるのか判定
		directAtPivot := operationAtNode(directRotation, atPivot)

		twoAtPivotParent := operationAtNode(twoNodeRotation, atPivotParent)
		twoAtPivot := operationAtNode(twoNodeRotation, atPivot)
		twoAtPivotChild := operationAtNode(twoNodeRotation, atPivotChild)
		threeAtPivotParent := operationAtNode(threeNodeRotation, atPivotParent)
		threeAtPivot := operationAtNode(threeNodeRotation, atPivot)
		threeAtPivotChild := operationAtNode(threeNodeRotation, atPivotChild)
		threeAtPivotGrandchild := operationAtNode(threeNodeRotation, atPivotGrandchild)

		/*
			rotationNone（挿入経路だけ表示。右挿入は左右反転）

			挿入前：
			  ancestor [Balanced]
			     /
			   parent [Balanced]
			     /
			    fake

			挿入後：
			  ancestor [Left]
			     /
			   parent [Left]
			     /
			  candidate [Balanced]

			pivotがないため、root側からcandidateの親まで挿入方向へ傾ける。
		*/
		// rotationNoneと、各操作完了後の挿入経路を更新
		updatePathBalance := shouldUpdatePathBalance(
			rebalanceActive,
			valid,
			atCandidate,
		) // balance更新区間内の実ノードかつcandidate以外なら1
		node.Balance = selectBalance(
			node.Balance,
			balanceFromDirection(right),
			updatePathBalance,
		) //新たに挿入される側（パスの進む側）にバランスが偏るはずなので更新（pivotやpivotchildは回転などでバランスの偏りが一定じゃないのでそれぞれ別の場所で更新）

		/*
			rotationDirect（pivotがLeft、右挿入の例。左右反転も同様）

			挿入前：
			    pivot [Left]
			    /    \
			existing  fake

			挿入後：
			    pivot [Balanced]
			    /    \
			existing  candidate [Balanced]

			低い側が伸びてもpivot全体の高さは変わらないため、
			pivotより上は更新せず、pivotをBalancedへ戻す。
		*/
		// rotationDirect：pivotをbalancedに戻し、その下の経路更新を開始
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			directAtPivot,
		) //Rotationdirectの際にpivotはバランスになる
		rebalanceActive = selectInt(
			rebalanceActive,
			1,
			directAtPivot,
		) // if directのpivot { active = 1 }; pivotより下を更新

		// rotationTwoNode：LLまたはRRの単回転
		/*
			LL（右単回転）

			回転前：
			          pivot
			         /     \
			    pivotChild  outside
			    /        \
			insertionSubtree middle
			    ...
			  candidate

			回転後：
			       pivotChild
			      /          \
			insertionSubtree  pivot
			    ...          /   \
			  candidate   middle outside

			pivotChild.Rightのmiddleをpivot.Leftへ移す。
			candidateはinsertionSubtreeのroot、またはその内部にある。

			RR（左単回転）

			回転前：
			      pivot
			     /     \
			outside     pivotChild
			            /        \
			         middle insertionSubtree
			                    ...
			                  candidate

			回転後：
			          pivotChild
			         /          \
			       pivot    insertionSubtree
			      /   \         ...
			outside middle    candidate

			pivotChild.Leftのmiddleをpivot.Rightへ移す。
			candidateはinsertionSubtreeのroot、またはその内部にある。
		*/
		parentLeftIsPivot := nodeIDsEqual(node.Left, metadata.pivotID)   // 親の左がpivotなら1
		parentRightIsPivot := nodeIDsEqual(node.Right, metadata.pivotID) // 親の右がpivotなら1
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotChildID,
			shouldReplaceParentLink(twoAtPivotParent, parentLeftIsPivot),
		) //親のノードの子ノードアドレスを変更
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotChildID,
			shouldReplaceParentLink(twoAtPivotParent, parentRightIsPivot),
		) //親のノードの子ノードアドレスを変更

		pivotTurnsRight := conditionNot(metadata.pivotDirection) // LLの右回転なら1
		pivotTurnsLeft := metadata.pivotDirection                // RRの左回転なら1

		//以下 pivotの子ノードアドレス更新
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotChildOppositeID,
			shouldApplyDirection(twoAtPivot, pivotTurnsRight),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotChildOppositeID,
			shouldApplyDirection(twoAtPivot, pivotTurnsLeft),
		)
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			twoAtPivot,
		)

		//以下 pivot childの子ノードアドレス更新
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotID,
			shouldApplyDirection(twoAtPivotChild, pivotTurnsLeft),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotID,
			shouldApplyDirection(twoAtPivotChild, pivotTurnsRight),
		)
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			twoAtPivotChild,
		)
		rebalanceActive = selectInt(
			rebalanceActive,
			1,
			twoAtPivotChild,
		) // if 単回転のpivot child { active = 1 }; 回転より下を更新

		/*
			rotationThreeNode：LRまたはRLの二重回転

			LR：
			      pivot                 pivotGrandchild
			     /     \                /             \
			pivotChild outside   -> pivotChild         pivot
			        \                 \                /
			   pivotGrandchild    grandchildLeft grandchildRight

			RL：
			pivot                              pivotGrandchild
			     \                            /             \
			     pivotChild       ->       pivot             pivotChild
			     /                            \              /
			pivotGrandchild              grandchildLeft grandchildRight
		*/
		parentLeftIsPivot = nodeIDsEqual(node.Left, metadata.pivotID)
		parentRightIsPivot = nodeIDsEqual(node.Right, metadata.pivotID)
		//parentの子はpivotからpivot childへ
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotGrandchildID,
			shouldReplaceParentLink(threeAtPivotParent, parentLeftIsPivot),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotGrandchildID,
			shouldReplaceParentLink(threeAtPivotParent, parentRightIsPivot),
		)

		//pivotのこはgrandchildに
		node.Left = selectNodeID(
			node.Left,
			metadata.grandchildRightID,
			shouldApplyDirection(
				threeAtPivot,
				conditionNot(metadata.pivotDirection),
			),
		) //LR
		node.Right = selectNodeID(
			node.Right,
			metadata.grandchildLeftID,
			shouldApplyDirection(threeAtPivot, metadata.pivotDirection),
		) //RL

		//pivotchildの子もgrandchildへ
		node.Left = selectNodeID(
			node.Left,
			metadata.grandchildRightID,
			shouldApplyDirection(threeAtPivotChild, metadata.pivotDirection),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.grandchildLeftID,
			shouldApplyDirection(
				threeAtPivotChild,
				conditionNot(metadata.pivotDirection),
			),
		)

		//pivot grandchildの子はpivotとpivot childに
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotChildID,
			shouldApplyDirection(
				threeAtPivotGrandchild,
				conditionNot(metadata.pivotDirection),
			),
		)
		node.Left = selectNodeID(
			node.Left,
			metadata.pivotID,
			shouldApplyDirection(threeAtPivotGrandchild, metadata.pivotDirection),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotID,
			shouldApplyDirection(
				threeAtPivotGrandchild,
				conditionNot(metadata.pivotDirection),
			),
		)
		node.Right = selectNodeID(
			node.Right,
			metadata.pivotChildID,
			shouldApplyDirection(threeAtPivotGrandchild, metadata.pivotDirection),
		)

		//grandchildがcadinateか判定
		grandchildIsCandidate := isInsertedGrandchild(
			metadata.pivotGrandchildID,
			candidateID,
			newInsertion,
		)

		//grandhildから傾いた方向を確認
		grandchildFollowsPivot := subtle.ConstantTimeEq(
			int32(metadata.grandchildDirection),
			int32(metadata.pivotDirection),
		)
		pivotTiltsOpposite := shouldTiltPivotOpposite(
			threeAtPivot,
			grandchildIsCandidate,
			grandchildFollowsPivot,
		)
		childTiltsPivot := shouldTiltChildTowardPivot(
			threeAtPivotChild,
			grandchildIsCandidate,
			grandchildFollowsPivot,
		)

		/*
			二重回転後のbalance

			回転  grandchild方向   pivot      pivotChild  grandchild
			LR    candidate自身    Balanced   Balanced    Balanced
			LR    Left             Right      Balanced    Balanced
			LR    Right            Balanced   Left        Balanced
			RL    candidate自身    Balanced   Balanced    Balanced
			RL    Right            Left       Balanced    Balanced
			RL    Left             Balanced   Right       Balanced
		*/
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			isThreeNodeRotationMember(
				threeAtPivot,
				threeAtPivotChild,
				threeAtPivotGrandchild,
			),
		)
		node.Balance = selectBalance(
			node.Balance,
			balanceFromDirection(conditionNot(metadata.pivotDirection)),
			pivotTiltsOpposite,
		)
		node.Balance = selectBalance(
			node.Balance,
			balanceFromDirection(metadata.pivotDirection),
			childTiltsPivot,
		)
		rebalanceActive = selectInt(
			rebalanceActive,
			1,
			threeAtPivotGrandchild,
		) // if 二重回転のpivot grandchild { active = 1 }; 回転より下を更新

		// 全操作共通：candidateへ到達したら経路更新を終了
		rebalanceActive = selectInt(
			rebalanceActive,
			0,
			atCandidate,
		) // if candidate { active = 0 }; 挿入位置で終了

		if err := t.store.Update(currentID, node); err != nil {
			return err
		}

		currentID = selectNodeID(t.fake, next, isNotNullAddress(next))
	}

	return err
}
