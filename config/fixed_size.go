// Package config contains compile-time settings shared across ENIGMAP layers.
package config

const (
	// KeySize is the exact byte length accepted by the AVL public key API.
	KeySize = 8

	// ValueSize is the serialized byte length of values stored in ORAM nodes.
	ValueSize = 8

	// KeyValueSize is the combined fixed payload size before AVL metadata.
	KeyValueSize = KeySize + ValueSize
)
