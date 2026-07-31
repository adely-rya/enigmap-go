package eoram

// PathServer is the path-level storage boundary used by Client.
//
// oram/pathoram.ORAMServer satisfies this interface. Its underlying
// oram/core.IO can therefore be wrapped with memory, file, cache, or encrypted
// IO without changing Client.
type OramServer[B any] interface {
	LeafCount() uint64
	ReadPath(position uint64) ([]B, error)
	WritePath(path []B, position uint64) error
}
