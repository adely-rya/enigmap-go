package eoram

import "github.com/adely-rya/oram/core"

// OramServer is the concrete Path ORAM server used by the ENIGMAP NodeStore.
type OramServer[T any, S core.IO[T]] struct {
	Storage S
}

// NewOramServer creates an ORAM server backed by storage.
func NewOramServer[T any, S core.IO[T]](storage S) OramServer[T, S] {
	return OramServer[T, S]{
		Storage: storage,
	}
}
