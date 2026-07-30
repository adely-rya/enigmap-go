package eoram

import (
	"errors"
	"testing"
)

type testCodec struct {
	size uint64
}

func (codec testCodec) Size() uint64 {
	return codec.size
}

func (codec testCodec) Encode(value uint64, destination []byte) error {
	return nil
}

func (codec testCodec) Decode(source []byte) (uint64, error) {
	return 0, nil
}

func TestDefaultUsesDefaultDataFile(t *testing.T) {
	store, err := Default[uint64](100, testCodec{size: 8})
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if store.capacity != 100 {
		t.Fatalf("capacity = %d, want 100", store.capacity)
	}
	if store.dataFile != DefaultDataFile {
		t.Fatalf("dataFile = %q, want %q", store.dataFile, DefaultDataFile)
	}
}

func TestNewUsesDataFile(t *testing.T) {
	store, err := New[uint64](100, testCodec{size: 8}, "/tmp/data.db")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store.dataFile != "/tmp/data.db" {
		t.Fatalf("dataFile = %q, want configured path", store.dataFile)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		new     func() (*ORAMClient[uint64], error)
		wantErr error
	}{
		{
			name: "zero capacity",
			new: func() (*ORAMClient[uint64], error) {
				return New[uint64](0, testCodec{size: 8}, "/tmp/data.db")
			},
			wantErr: ErrInvalidCapacity,
		},
		{
			name: "nil codec",
			new: func() (*ORAMClient[uint64], error) {
				return New[uint64](100, nil, "/tmp/data.db")
			},
			wantErr: ErrNilValueCodec,
		},
		{
			name: "zero value size",
			new: func() (*ORAMClient[uint64], error) {
				return New[uint64](100, testCodec{}, "/tmp/data.db")
			},
			wantErr: ErrInvalidValueSize,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.new()
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
