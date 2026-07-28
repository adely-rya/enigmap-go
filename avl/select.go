package avl

func selectNodeID(current, candidate NodeID, condition int) NodeID {
	mask := uint64(0) - uint64(condition&1)
	return NodeID(
		(uint64(current) &^ mask) |
			(uint64(candidate) & mask),
	)
}

func nodeIDPresent(id NodeID) int {
	value := uint64(id)
	return int((value | (0 - value)) >> 63)
}
