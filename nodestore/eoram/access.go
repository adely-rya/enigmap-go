package eoram

import "github.com/adely-rya/enigmap-go/avl"

// AccessState identifies the one access opened by BeginAccess and completed by
// FinishAccess.
type AccessState struct {
	Address   avl.NodeID
	SavedPath avl.Position
	Open      bool
}
