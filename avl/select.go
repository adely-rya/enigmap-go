package avl

import "crypto/subtle"

func selectInt(current, candidate, condition int) int {
	// conditionが0ならcurrent、1ならcandidate
	return subtle.ConstantTimeSelect(condition&1, candidate, current)
}

func selectBalance(current, candidate Balance, condition int) Balance {
	// conditionが0ならcurrent、1ならcandidate
	return Balance(selectInt(int(current), int(candidate), condition))
}

func selectRotationType(
	current,
	candidate rotationType,
	condition int,
) rotationType {
	// conditionが0ならcurrent、1ならcandidate
	return rotationType(selectInt(int(current), int(candidate), condition))
}

func selectValue[V any](current, candidate V, condition int) V {
	if condition&1 == 1 {
		return candidate
	}
	return current
}
