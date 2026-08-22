package avl

import "crypto/subtle"

func selectInt(current, candidate, condition int) int {
	return subtle.ConstantTimeSelect(condition&1, candidate, current)
}

func selectBalance(current, candidate Balance, condition int) Balance {
	return Balance(selectInt(int(current), int(candidate), condition))
}

func selectRotationType(
	current,
	candidate rotationType,
	condition int,
) rotationType {
	return rotationType(selectInt(int(current), int(candidate), condition))
}

func rotationTypesEqual(a, b rotationType) int {
	return subtle.ConstantTimeByteEq(byte(a), byte(b))
}

func balanceFromDirection(right int) Balance {
	return selectBalance(BalanceLeft, BalanceRight, right)
}

func selectValue[V any](current, candidate V, condition int) V {
	if condition&1 == 1 {
		return candidate
	}
	return current
}
