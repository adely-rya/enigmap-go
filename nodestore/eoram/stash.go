package eoram

// StashedBlock adds client-only eviction state to an ORAM block.
type StashedBlock struct {
	Block  Block
	Sticky uint8
}

// Stash holds blocks read from ORAM paths.
type Stash struct {
	blocks []StashedBlock
}
