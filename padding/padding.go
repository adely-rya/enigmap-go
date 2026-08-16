package padding

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/adely-rya/oram/core"
)

const BlockSize = 4 * 1024

var (
	ErrDataTooLarge = errors.New("paddingio: serialized value exceeds 4 KiB")
	ErrOutOfRange   = errors.New("paddingio: index out of range")
)

type fixedSerializable interface {
	core.Serializable
	Size() uint64
}

// PaddingIO stores each value in exactly one 4 KiB file block.
// Serialized values are zero-padded on write and the padding is removed on read.
type PaddingIO[T fixedSerializable] struct {
	file *os.File
	size uint64
}

func New[T fixedSerializable](path string, size uint64) (*PaddingIO[T], error) {
	var zero T
	if zero.Size() > BlockSize {
		return nil, fmt.Errorf("%w: got=%d max=%d", ErrDataTooLarge, zero.Size(), BlockSize)
	}
	if size > uint64(math.MaxInt64/BlockSize) {
		return nil, fmt.Errorf("paddingio: file size overflow: records=%d", size)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("paddingio: open file: %w", err)
	}
	if err := file.Truncate(int64(size) * BlockSize); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("paddingio: resize file: %w", err)
	}

	return &PaddingIO[T]{file: file, size: size}, nil
}

func (storage *PaddingIO[T]) Size() uint64 {
	return storage.size
}

func (storage *PaddingIO[T]) Read(index uint64) (T, error) {
	var zero T
	if index >= storage.size {
		return zero, fmt.Errorf("%w: index=%d size=%d", ErrOutOfRange, index, storage.size)
	}

	var block [BlockSize]byte
	if _, err := storage.file.ReadAt(block[:], int64(index)*BlockSize); err != nil {
		return zero, fmt.Errorf("paddingio: read block %d: %w", index, err)
	}

	contentSize := zero.Size()
	decoded, err := zero.Deserialize(block[:int(contentSize)])
	if err != nil {
		return zero, fmt.Errorf("paddingio: deserialize block %d: %w", index, err)
	}
	value, ok := decoded.(T)
	if !ok {
		return zero, fmt.Errorf("paddingio: deserialized block %d has unexpected type", index)
	}
	return value, nil
}

func (storage *PaddingIO[T]) Write(value T, index uint64) error {
	if index >= storage.size {
		return fmt.Errorf("%w: index=%d size=%d", ErrOutOfRange, index, storage.size)
	}

	encoded, err := value.Serialize()
	if err != nil {
		return fmt.Errorf("paddingio: serialize block %d: %w", index, err)
	}
	want := value.Size()
	if uint64(len(encoded)) != want {
		return fmt.Errorf(
			"paddingio: serialized size mismatch: got=%d want=%d",
			len(encoded),
			want,
		)
	}
	if len(encoded) > BlockSize {
		return fmt.Errorf("%w: got=%d max=%d", ErrDataTooLarge, len(encoded), BlockSize)
	}

	var block [BlockSize]byte
	copy(block[:], encoded)
	n, err := storage.file.WriteAt(block[:], int64(index)*BlockSize)
	if err != nil {
		return fmt.Errorf("paddingio: write block %d: %w", index, err)
	}
	if n != BlockSize {
		return io.ErrShortWrite
	}
	return nil
}

func (storage *PaddingIO[T]) Close() error {
	return storage.file.Close()
}
