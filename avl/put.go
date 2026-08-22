package avl

import (
	"crypto/subtle"
)

type rotationType uint8

const (
	rotationNone rotationType = iota
	rotationDirect
	rotationTwoNode
	rotationThreeNode
)

// putMetadata carries state between the fixed-depth Put passes.
type putMetadata struct {
	insertionParentID    Pointer
	insertionRight       int
	keyExists            int
	existingNodeID       Pointer
	pivotID              Pointer
	pivotBalance         Balance
	pivotParentID        Pointer
	pivotChildID         Pointer
	pivotChildOppositeID Pointer
	pivotGrandchildID    Pointer
	grandchildLeftID     Pointer
	grandchildRightID    Pointer
	pivotDirection       int
	childDirection       int
	grandchildDirection  int
	rotation             rotationType
}

func refreshPointerLeaf(pointer, accessed Pointer) Pointer {
	return selectPointer(
		pointer,
		accessed,
		pointerAddressesEqual(pointer, accessed),
	)
}

func (metadata *putMetadata) refreshPointerLeaves(accessed Pointer) {
	metadata.insertionParentID = refreshPointerLeaf(metadata.insertionParentID, accessed)
	metadata.existingNodeID = refreshPointerLeaf(metadata.existingNodeID, accessed)
	metadata.pivotID = refreshPointerLeaf(metadata.pivotID, accessed)
	metadata.pivotParentID = refreshPointerLeaf(metadata.pivotParentID, accessed)
	metadata.pivotChildID = refreshPointerLeaf(metadata.pivotChildID, accessed)
	metadata.pivotChildOppositeID = refreshPointerLeaf(metadata.pivotChildOppositeID, accessed)
	metadata.pivotGrandchildID = refreshPointerLeaf(metadata.pivotGrandchildID, accessed)
	metadata.grandchildLeftID = refreshPointerLeaf(metadata.grandchildLeftID, accessed)
	metadata.grandchildRightID = refreshPointerLeaf(metadata.grandchildRightID, accessed)
}

func pointerWithPlannedLeaf(
	pointer Pointer,
	addresses []NodeID,
	leaves []ORAMPosition,
) Pointer {
	found := 0
	for index := 0; index < len(addresses); index++ {
		match := nodeIDsEqual(pointer.Address, addresses[index]) & conditionNot(found)
		pointer.Position = ORAMPosition(selectNodeID(
			NodeID(pointer.Position),
			NodeID(leaves[index]),
			match,
		))
		found |= match
	}

	return pointer
}

func (metadata *putMetadata) applyPlannedLeaves(
	addresses []NodeID,
	leaves []ORAMPosition,
) {
	metadata.insertionParentID = pointerWithPlannedLeaf(metadata.insertionParentID, addresses, leaves)
	metadata.existingNodeID = pointerWithPlannedLeaf(metadata.existingNodeID, addresses, leaves)
	metadata.pivotID = pointerWithPlannedLeaf(metadata.pivotID, addresses, leaves)
	metadata.pivotParentID = pointerWithPlannedLeaf(metadata.pivotParentID, addresses, leaves)
	metadata.pivotChildID = pointerWithPlannedLeaf(metadata.pivotChildID, addresses, leaves)
	metadata.pivotChildOppositeID = pointerWithPlannedLeaf(metadata.pivotChildOppositeID, addresses, leaves)
	metadata.pivotGrandchildID = pointerWithPlannedLeaf(metadata.pivotGrandchildID, addresses, leaves)
	metadata.grandchildLeftID = pointerWithPlannedLeaf(metadata.grandchildLeftID, addresses, leaves)
	metadata.grandchildRightID = pointerWithPlannedLeaf(metadata.grandchildRightID, addresses, leaves)
}

// Conditional helpers operate on normalized 0/1 values.
func conditionNot(condition int) int {
	return condition ^ 1
}

func conditionsBoth(first, second int) int {
	return first & second
}

func conditionsAll3(first, second, third int) int {
	return first & second & third
}

func conditionsAll4(first, second, third, fourth int) int {
	return first & second & third & fourth
}

func conditionsAny3(first, second, third int) int {
	return first | second | third
}

func rememberCondition(previous, current int) int {
	return previous | current
}

func matchingRealNode(keysEqual, valid int) int {
	return conditionsBoth(keysEqual, valid)
}

func directionForRealNode(right, valid int) int {
	return conditionsBoth(right, valid)
}

func currentIsStoredNode(current, stored Pointer, storedNodeExists int) int {
	return conditionsBoth(pointerAddressesEqual(current, stored), storedNodeExists)
}

func isPivotCandidate(valid, balanced int) int {
	return conditionsBoth(valid, conditionNot(balanced))
}

func isFirstInsertionPoint(nextIsFake, currentIsFake, keyExists, insertionPointFound int) int {
	return conditionsAll4(
		nextIsFake,
		conditionNot(currentIsFake),
		conditionNot(keyExists),
		conditionNot(insertionPointFound),
	)
}

func candidateBecomesPivotGrandchild(currentIsPivotChild, foundInsertionPoint int) int {
	return conditionsBoth(currentIsPivotChild, foundInsertionPoint)
}

func shouldInsertLeft(foundInsertionPoint, right int) int {
	return conditionsBoth(foundInsertionPoint, conditionNot(right))
}

func shouldInsertRight(foundInsertionPoint, right int) int {
	return conditionsBoth(foundInsertionPoint, right)
}

func shouldClassifyRebalance(pivotExists, keyExists, insertionPointFound int) int {
	return conditionsAll3(pivotExists, conditionNot(keyExists), insertionPointFound)
}

func isDirectRebalance(shouldRebalance, pivotFollowsBalance int) int {
	return conditionsBoth(shouldRebalance, conditionNot(pivotFollowsBalance))
}

func isTwoNodeRotation(shouldRebalance, pivotFollowsBalance, childFollowsPivot int) int {
	return conditionsAll3(shouldRebalance, pivotFollowsBalance, childFollowsPivot)
}

func isThreeNodeRotation(shouldRebalance, pivotFollowsBalance, childFollowsPivot int) int {
	return conditionsAll3(
		shouldRebalance,
		pivotFollowsBalance,
		conditionNot(childFollowsPivot),
	)
}

func isNewInsertion(keyExists, insertionPointFound int) int {
	return conditionsBoth(conditionNot(keyExists), insertionPointFound)
}

func shouldStartPathBalance(newInsertion, rotationIsNone int) int {
	return conditionsBoth(newInsertion, rotationIsNone)
}

func operationAtNode(operationSelected, atNode int) int {
	return conditionsBoth(operationSelected, atNode)
}

func shouldUpdatePathBalance(rebalanceActive, valid, atCandidate int) int {
	return conditionsAll3(rebalanceActive, valid, conditionNot(atCandidate))
}

func shouldReplaceParentLink(atRotationParent, childIsPivot int) int {
	return conditionsBoth(atRotationParent, childIsPivot)
}

func shouldApplyDirection(atRotationNode, direction int) int {
	return conditionsBoth(atRotationNode, direction)
}

func isInsertedGrandchild(pivotGrandchild, candidate Pointer, newInsertion int) int {
	return conditionsBoth(pointerAddressesEqual(pivotGrandchild, candidate), newInsertion)
}

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

func isThreeNodeRotationMember(threeAtPivot, threeAtPivotChild, threeAtPivotGrandchild int) int {
	return conditionsAny3(threeAtPivot, threeAtPivotChild, threeAtPivotGrandchild)
}

// Put performs a fixed-depth insertion or update followed by rebalancing.
func (t *Tree[V]) Put(key []byte, value V) error {
	defer t.store.UncacheAll()

	insertKey := NewKey(key)

	candidate := Node[V]{
		Key:     insertKey,
		Value:   value,
		Balance: BalanceBalanced,
		Left:    t.fake,
		Right:   t.fake,
		Valid:   1,
	}

	// Allocate independently of whether the key already exists.
	candidatePointer, err := t.store.Allocate(candidate)
	if err != nil {
		return err
	}

	metadata := putMetadata{
		rotation: rotationNone,
	}

	currentID := t.root
	currentNewPosition, err := t.store.NewPosition()
	if err != nil {
		return err
	}
	t.root.Position = currentNewPosition
	parentID := Pointer{}
	insertionPointFound := 0
	var firstPassAddressBuffer [maxAVLDepth]NodeID
	firstPassAddresses := firstPassAddressBuffer[:t.maxDepth]

	for depth := 0; depth < t.maxDepth; depth++ {
		firstPassAddresses[depth] = currentID.Address
		nextNewPosition, err := t.store.NewPosition()
		if err != nil {
			return err
		}

		node, token, err := t.store.BeginAccess(currentID, currentNewPosition)
		if err != nil {
			return err
		}
		currentNew := pointerWithPosition(currentID, currentNewPosition)
		metadata.refreshPointerLeaves(currentNew)
		candidatePointer = refreshPointerLeaf(candidatePointer, currentNew)
		t.fake = selectPointer(
			t.fake,
			currentNew,
			pointerAddressesEqual(currentID, t.fake),
		)

		comparison := CompareKey(insertKey, node.Key)
		equal := subtle.ConstantTimeEq(int32(comparison), 0)
		right := subtle.ConstantTimeEq(int32(comparison), 1)
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)

		match := matchingRealNode(equal, valid)
		right = directionForRealNode(right, valid)

		metadata.existingNodeID = selectPointer(
			metadata.existingNodeID,
			currentNew,
			match,
		)
		metadata.keyExists = rememberCondition(metadata.keyExists, match)
		node.Value = selectValue(node.Value, value, match)

		next := selectPointer(node.Left, node.Right, right)
		nextNew := pointerWithPosition(next, nextNewPosition)
		node.Left = selectPointer(node.Left, nextNew, conditionNot(right))
		node.Right = selectPointer(node.Right, nextNew, right)

		pivotExists := isNotNullPointer(metadata.pivotID)
		currentIsPivotChild := currentIsStoredNode(
			currentID,
			metadata.pivotChildID,
			pivotExists,
		)
		metadata.childDirection = selectInt(
			metadata.childDirection,
			right,
			currentIsPivotChild,
		)
		oppositeChild := selectPointer(
			node.Right,
			node.Left,
			right,
		)
		metadata.pivotChildOppositeID = selectPointer(
			metadata.pivotChildOppositeID,
			oppositeChild,
			currentIsPivotChild,
		)
		metadata.pivotGrandchildID = selectPointer(
			metadata.pivotGrandchildID,
			next,
			currentIsPivotChild,
		)

		grandchildExists := isNotNullPointer(metadata.pivotGrandchildID)
		currentIsPivotGrandchild := currentIsStoredNode(
			currentID,
			metadata.pivotGrandchildID,
			grandchildExists,
		)
		metadata.grandchildLeftID = selectPointer(
			metadata.grandchildLeftID,
			node.Left,
			currentIsPivotGrandchild,
		)
		metadata.grandchildRightID = selectPointer(
			metadata.grandchildRightID,
			node.Right,
			currentIsPivotGrandchild,
		)
		metadata.grandchildDirection = selectInt(
			metadata.grandchildDirection,
			right,
			currentIsPivotGrandchild,
		)

		balanced := subtle.ConstantTimeByteEq(
			byte(node.Balance),
			byte(BalanceBalanced),
		)
		pivotCandidate := isPivotCandidate(valid, balanced)
		metadata.pivotID = selectPointer(
			metadata.pivotID,
			currentNew,
			pivotCandidate,
		)
		metadata.pivotParentID = selectPointer(
			metadata.pivotParentID,
			parentID,
			pivotCandidate,
		)
		metadata.pivotChildID = selectPointer(
			metadata.pivotChildID,
			next,
			pivotCandidate,
		)
		metadata.pivotDirection = selectInt(
			metadata.pivotDirection,
			right,
			pivotCandidate,
		)
		metadata.pivotBalance = selectBalance(
			metadata.pivotBalance,
			node.Balance,
			pivotCandidate,
		)
		metadata.pivotGrandchildID = selectPointer(
			metadata.pivotGrandchildID,
			Pointer{},
			pivotCandidate,
		)

		nextIsFake := pointerAddressesEqual(next, t.fake)
		currentIsFake := pointerAddressesEqual(currentID, t.fake)
		foundInsertionPoint := isFirstInsertionPoint(
			nextIsFake,
			currentIsFake,
			metadata.keyExists,
			insertionPointFound,
		)

		metadata.insertionParentID = selectPointer(
			metadata.insertionParentID,
			currentNew,
			foundInsertionPoint,
		)
		metadata.insertionRight = selectInt(
			metadata.insertionRight,
			right,
			foundInsertionPoint,
		)
		candidateIsPivotGrandchild := candidateBecomesPivotGrandchild(
			currentIsPivotChild,
			foundInsertionPoint,
		)
		metadata.pivotGrandchildID = selectPointer(
			metadata.pivotGrandchildID,
			candidatePointer,
			candidateIsPivotGrandchild,
		)
		metadata.grandchildLeftID = selectPointer(
			metadata.grandchildLeftID,
			t.fake,
			candidateIsPivotGrandchild,
		)
		metadata.grandchildRightID = selectPointer(
			metadata.grandchildRightID,
			t.fake,
			candidateIsPivotGrandchild,
		)
		insertionPointFound = rememberCondition(insertionPointFound, foundInsertionPoint)

		insertLeft := shouldInsertLeft(foundInsertionPoint, right)
		insertRight := shouldInsertRight(foundInsertionPoint, right)
		node.Left = selectPointer(node.Left, candidatePointer, insertLeft)
		node.Right = selectPointer(node.Right, candidatePointer, insertRight)

		if err := t.store.FinishAccess(token, node); err != nil {
			return err
		}
		next = selectPointer(
			next,
			t.fake,
			pointerAddressesEqual(next, t.fake),
		)

		parentID = selectPointer(
			parentID,
			currentNew,
			conditionNot(currentIsFake),
		)
		currentID = selectPointer(t.fake, next, isNotNullPointer(next))
		currentNewPosition = nextNewPosition
	}

	pivotExists := isNotNullPointer(metadata.pivotID)
	shouldRebalance := shouldClassifyRebalance(
		pivotExists,
		metadata.keyExists,
		insertionPointFound,
	)
	pivotFollowsBalance := subtle.ConstantTimeByteEq(
		byte(metadata.pivotBalance),
		byte(metadata.pivotDirection),
	)
	childFollowsPivot := subtle.ConstantTimeEq(
		int32(metadata.childDirection),
		int32(metadata.pivotDirection),
	)

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

	err = t.store.FreeMaybe(candidatePointer, metadata.keyExists)
	if err != nil {
		return err
	}

	var secondPassLeafBuffer [maxAVLDepth + 1]ORAMPosition
	secondPassLeaves := secondPassLeafBuffer[:t.maxDepth+1]
	for depth := 0; depth < len(secondPassLeaves); depth++ {
		secondPassLeaves[depth], err = t.store.NewPosition()
		if err != nil {
			return err
		}
	}
	candidateSecondLeaf, err := t.store.NewPosition()
	if err != nil {
		return err
	}
	metadata.applyPlannedLeaves(firstPassAddresses, secondPassLeaves[:t.maxDepth])
	candidatePointer.Position = candidateSecondLeaf

	currentID = t.root
	currentNewPosition = secondPassLeaves[0]
	t.root.Position = currentNewPosition
	newInsertion := isNewInsertion(metadata.keyExists, insertionPointFound)
	rotationIsNone := rotationTypesEqual(metadata.rotation, rotationNone)
	rebalanceActive := shouldStartPathBalance(newInsertion, rotationIsNone)

	for depth := 0; depth < t.maxDepth; depth++ {
		currentNewPosition = ORAMPosition(selectNodeID(
			NodeID(secondPassLeaves[depth]),
			NodeID(candidateSecondLeaf),
			pointerAddressesEqual(currentID, candidatePointer),
		))

		node, token, err := t.store.BeginAccess(currentID, currentNewPosition)
		if err != nil {
			return err
		}
		currentNew := pointerWithPosition(currentID, currentNewPosition)
		metadata.refreshPointerLeaves(currentNew)
		candidatePointer = refreshPointerLeaf(candidatePointer, currentNew)
		t.fake = selectPointer(
			t.fake,
			currentNew,
			pointerAddressesEqual(currentID, t.fake),
		)

		comparison := CompareKey(insertKey, node.Key)
		right := subtle.ConstantTimeEq(int32(comparison), 1)
		valid := subtle.ConstantTimeByteEq(node.Valid, 1)
		right = directionForRealNode(right, valid)
		next := selectPointer(node.Left, node.Right, right)
		nextNewPosition := ORAMPosition(selectNodeID(
			NodeID(secondPassLeaves[depth+1]),
			NodeID(candidateSecondLeaf),
			pointerAddressesEqual(next, candidatePointer),
		))
		nextNew := pointerWithPosition(next, nextNewPosition)
		node.Left = selectPointer(node.Left, nextNew, conditionNot(right))
		node.Right = selectPointer(node.Right, nextNew, right)

		atPivot := currentIsStoredNode(currentID, metadata.pivotID, pivotExists)
		atPivotParent := currentIsStoredNode(currentID, metadata.pivotParentID, pivotExists)
		atPivotChild := currentIsStoredNode(currentID, metadata.pivotChildID, pivotExists)
		atPivotGrandchild := currentIsStoredNode(
			currentID,
			metadata.pivotGrandchildID,
			pivotExists,
		)
		atCandidate := currentIsStoredNode(currentID, candidatePointer, newInsertion)

		directRotation := rotationTypesEqual(
			metadata.rotation,
			rotationDirect,
		)
		directAtPivot := operationAtNode(directRotation, atPivot)

		twoAtPivotParent := operationAtNode(twoNodeRotation, atPivotParent)
		twoAtPivot := operationAtNode(twoNodeRotation, atPivot)
		twoAtPivotChild := operationAtNode(twoNodeRotation, atPivotChild)
		threeAtPivotParent := operationAtNode(threeNodeRotation, atPivotParent)
		threeAtPivot := operationAtNode(threeNodeRotation, atPivot)
		threeAtPivotChild := operationAtNode(threeNodeRotation, atPivotChild)
		threeAtPivotGrandchild := operationAtNode(threeNodeRotation, atPivotGrandchild)

		// Path balance update.
		updatePathBalance := shouldUpdatePathBalance(
			rebalanceActive,
			valid,
			atCandidate,
		)
		node.Balance = selectBalance(
			node.Balance,
			balanceFromDirection(right),
			updatePathBalance,
		)

		// Direct rebalance.
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			directAtPivot,
		)
		rebalanceActive = selectInt(
			rebalanceActive,
			1,
			directAtPivot,
		)

		// Single rotation.
		parentLeftIsPivot := pointerAddressesEqual(node.Left, metadata.pivotID)
		parentRightIsPivot := pointerAddressesEqual(node.Right, metadata.pivotID)
		node.Left = selectPointer(
			node.Left,
			metadata.pivotChildID,
			shouldReplaceParentLink(twoAtPivotParent, parentLeftIsPivot),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.pivotChildID,
			shouldReplaceParentLink(twoAtPivotParent, parentRightIsPivot),
		)

		pivotTurnsRight := conditionNot(metadata.pivotDirection)
		pivotTurnsLeft := metadata.pivotDirection

		node.Left = selectPointer(
			node.Left,
			metadata.pivotChildOppositeID,
			shouldApplyDirection(twoAtPivot, pivotTurnsRight),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.pivotChildOppositeID,
			shouldApplyDirection(twoAtPivot, pivotTurnsLeft),
		)
		node.Balance = selectBalance(
			node.Balance,
			BalanceBalanced,
			twoAtPivot,
		)

		node.Left = selectPointer(
			node.Left,
			metadata.pivotID,
			shouldApplyDirection(twoAtPivotChild, pivotTurnsLeft),
		)
		node.Right = selectPointer(
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
		)

		// Double rotation.
		parentLeftIsPivot = pointerAddressesEqual(node.Left, metadata.pivotID)
		parentRightIsPivot = pointerAddressesEqual(node.Right, metadata.pivotID)
		node.Left = selectPointer(
			node.Left,
			metadata.pivotGrandchildID,
			shouldReplaceParentLink(threeAtPivotParent, parentLeftIsPivot),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.pivotGrandchildID,
			shouldReplaceParentLink(threeAtPivotParent, parentRightIsPivot),
		)

		node.Left = selectPointer(
			node.Left,
			metadata.grandchildRightID,
			shouldApplyDirection(
				threeAtPivot,
				conditionNot(metadata.pivotDirection),
			),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.grandchildLeftID,
			shouldApplyDirection(threeAtPivot, metadata.pivotDirection),
		)

		node.Left = selectPointer(
			node.Left,
			metadata.grandchildRightID,
			shouldApplyDirection(threeAtPivotChild, metadata.pivotDirection),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.grandchildLeftID,
			shouldApplyDirection(
				threeAtPivotChild,
				conditionNot(metadata.pivotDirection),
			),
		)

		node.Left = selectPointer(
			node.Left,
			metadata.pivotChildID,
			shouldApplyDirection(
				threeAtPivotGrandchild,
				conditionNot(metadata.pivotDirection),
			),
		)
		node.Left = selectPointer(
			node.Left,
			metadata.pivotID,
			shouldApplyDirection(threeAtPivotGrandchild, metadata.pivotDirection),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.pivotID,
			shouldApplyDirection(
				threeAtPivotGrandchild,
				conditionNot(metadata.pivotDirection),
			),
		)
		node.Right = selectPointer(
			node.Right,
			metadata.pivotChildID,
			shouldApplyDirection(threeAtPivotGrandchild, metadata.pivotDirection),
		)

		grandchildIsCandidate := isInsertedGrandchild(
			metadata.pivotGrandchildID,
			candidatePointer,
			newInsertion,
		)

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
		)

		rebalanceActive = selectInt(
			rebalanceActive,
			0,
			atCandidate,
		)

		if err := t.store.FinishAccess(token, node); err != nil {
			return err
		}
		next = selectPointer(
			next,
			t.fake,
			pointerAddressesEqual(next, t.fake),
		)

		currentID = selectPointer(t.fake, next, isNotNullPointer(next))
		currentNewPosition = nextNewPosition
	}

	return err
}
