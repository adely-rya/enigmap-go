package eoram

import "github.com/adely-rya/oram/core"

// OramServer is the concrete Path ORAM server used by the ENIGMAP NodeStore.
type OramServer[T any, S core.IO[T]] struct {
	Storage S
}
