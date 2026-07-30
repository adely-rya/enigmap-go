package eoram

import (
	"errors"
	"testing"

	"github.com/adely-rya/enigmap-go/avl"
	"github.com/adely-rya/enigmap-go/omap"
)

func TestDefaultUsesDefaultDataFile(t *testing.T) {
	client, err := Default(100)
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	if client.capacity != 100 {
		t.Fatalf("capacity = %d, want 100", client.capacity)
	}
	if client.dataFile != DefaultDataFile {
		t.Fatalf("dataFile = %q, want %q", client.dataFile, DefaultDataFile)
	}
}

func TestNewUsesDataFile(t *testing.T) {
	client, err := New(100, "/tmp/data.db")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if client.dataFile != "/tmp/data.db" {
		t.Fatalf("dataFile = %q, want configured path", client.dataFile)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	_, err := New(0, "/tmp/data.db")
	if !errors.Is(err, ErrInvalidCapacity) {
		t.Fatalf("New() error = %v, want %v", err, ErrInvalidCapacity)
	}
}

func TestNodeStoreOperationsReportNotImplemented(t *testing.T) {
	client, err := New(100, "/tmp/data.db")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	node := avl.Node[omap.Value]{}
	if _, err := client.Allocate(node); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Allocate() error = %v, want %v", err, ErrNotImplemented)
	}
	if _, err := client.Load(1); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Load() error = %v, want %v", err, ErrNotImplemented)
	}
	if err := client.Update(1, node); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Update() error = %v, want %v", err, ErrNotImplemented)
	}
	if err := client.FreeMaybe(1, 0); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("FreeMaybe() error = %v, want %v", err, ErrNotImplemented)
	}
}
