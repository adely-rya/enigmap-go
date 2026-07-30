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

func TestNewUsesSimpleDefaults(t *testing.T) {
	store, err := New[uint64](100, testCodec{size: 8})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store.maxNodes != 100 {
		t.Fatalf("maxNodes = %d, want 100", store.maxNodes)
	}
	if store.config.levelsPerPack != defaultLevelsPerPack {
		t.Fatalf(
			"levelsPerPack = %d, want %d",
			store.config.levelsPerPack,
			defaultLevelsPerPack,
		)
	}
	if store.config.cacheCapacity != defaultCacheCapacity {
		t.Fatalf(
			"cacheCapacity = %d, want %d",
			store.config.cacheCapacity,
			defaultCacheCapacity,
		)
	}
	if store.config.dataFile != defaultDataFile {
		t.Fatalf("dataFile = %q, want %q", store.config.dataFile, defaultDataFile)
	}
}

func TestNewAppliesOptions(t *testing.T) {
	store, err := New[uint64](
		100,
		testCodec{size: 8},
		WithLevelsPerPack(2),
		WithCacheCapacity(16),
		WithDataFile("/dev/shm/enigmap/data.db"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store.config.levelsPerPack != 2 {
		t.Fatalf("levelsPerPack = %d, want 2", store.config.levelsPerPack)
	}
	if store.config.cacheCapacity != 16 {
		t.Fatalf("cacheCapacity = %d, want 16", store.config.cacheCapacity)
	}
	if store.config.dataFile != "/dev/shm/enigmap/data.db" {
		t.Fatalf("dataFile = %q, want configured path", store.config.dataFile)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		new     func() (*Store[uint64], error)
		wantErr error
	}{
		{
			name: "zero max nodes",
			new: func() (*Store[uint64], error) {
				return New[uint64](0, testCodec{size: 8})
			},
			wantErr: ErrInvalidMaxNodes,
		},
		{
			name: "nil codec",
			new: func() (*Store[uint64], error) {
				return New[uint64](100, nil)
			},
			wantErr: ErrNilValueCodec,
		},
		{
			name: "zero value size",
			new: func() (*Store[uint64], error) {
				return New[uint64](100, testCodec{})
			},
			wantErr: ErrInvalidValueSize,
		},
		{
			name: "zero levels per pack",
			new: func() (*Store[uint64], error) {
				return New[uint64](
					100,
					testCodec{size: 8},
					WithLevelsPerPack(0),
				)
			},
			wantErr: ErrInvalidLevelsPerPack,
		},
		{
			name: "zero cache capacity",
			new: func() (*Store[uint64], error) {
				return New[uint64](
					100,
					testCodec{size: 8},
					WithCacheCapacity(0),
				)
			},
			wantErr: ErrInvalidCacheCapacity,
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
