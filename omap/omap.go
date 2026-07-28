package omap

type Store[V any] interface {
	Get(key []byte) (V, bool, error)
	Put(key []byte, value V) error
}
