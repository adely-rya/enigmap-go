package eoram

import "github.com/adely-rya/oram/core"

// ORAMServer reads and writes ORAM values through core.IO.
type ORAMServer[T any, S core.IO[T]] struct {
	Storage S
}
