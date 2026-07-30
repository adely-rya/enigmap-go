package codec

// ValueCodec converts a value to and from a fixed-size byte representation.
// The fixed size allows eoram to keep every ORAM block and file record equal
// in length.
type ValueCodec[V any] interface {
	Size() uint64
	Encode(V, []byte) error
	Decode([]byte) (V, error)
}
