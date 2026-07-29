package avl

func selectValue[V any](current, candidate V, condition int) V {
	if condition&1 == 1 {
		return candidate
	}
	return current
}
