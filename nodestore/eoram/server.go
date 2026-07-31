package eoram

// StorageIO is the bucket storage boundary used by OramServer.
//
// Its method set matches oram/core.IO so memory, file, cache, and encrypted IO
// implementations can be inserted without changing the server.
type StorageIO[B any] interface {
	Read(index uint64) (B, error)
	Write(value B, index uint64) error
	Size() uint64
}

// OramServer is the concrete Path ORAM server used by the ENIGMAP NodeStore.
type OramServer[B any] struct {
	Storage StorageIO[B]
}
