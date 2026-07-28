package omap

// ValueSelector selects candidate when condition is 1 and current when it is
// 0. Implementations must avoid branches and memory accesses that depend on
// condition.
type ValueSelector[V any] interface {
	Select(current, candidate V, condition int) V
}

type ValueSelectorFunc[V any] func(current, candidate V, condition int) V

func (f ValueSelectorFunc[V]) Select(current, candidate V, condition int) V {
	return f(current, candidate, condition)
}
