package eoram

import "github.com/adely-rya/oram/core"

// OramServer is the concrete Path ORAM server used by the ENIGMAP NodeStore.
type OramServer[B any] struct {
	Storage core.IO[B]
}
