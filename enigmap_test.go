package enigmap

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/nodestore/eoram"
	"github.com/adely-rya/oram/storage"
)

type selectableValue uint64

func (value selectableValue) Select(
	candidate selectableValue,
	condition int,
) selectableValue {
	if condition == 1 {
		return candidate
	}

	return value
}

func TestNewORAMReportsMissingNodeStore(t *testing.T) {
	store := storage.NewMemoryStorage[eoram.Bucket[selectableValue]](7)

	got, err := NewORAM[selectableValue](store, 3, 5)

	if got != nil {
		t.Fatalf("store = %v, want nil", got)
	}
	if !errors.Is(err, ErrORAMNodeStoreNotImplemented) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			ErrORAMNodeStoreNotImplemented,
		)
	}
}
