package eoram

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/adely-rya/enigmap-go/avl"
)

func TestFixedValueBlockWordLayout(t *testing.T) {
	size := unsafe.Sizeof(fixedValueBlock{})
	if size%8 != 0 {
		t.Fatalf("fixed block size %d is not word aligned", size)
	}
	if got, want := fixedValueBlockWordCount, int(size/8); got != want {
		t.Fatalf("word count = %d, want %d", got, want)
	}
	if hasPointers(reflect.TypeOf(fixedValueBlock{})) {
		t.Fatal("fixed block contains pointers and cannot be selected as raw words")
	}
}

func TestSelectFixedValueBlockWords(t *testing.T) {
	current := fixedValueBlock{
		BlockAddress: 1,
		Leaf:         2,
		Data: NodeBlock[FixedValue]{Node: avl.Node[FixedValue]{
			Key:   avl.NewKey(fixedTestKey("current!")),
			Value: FixedValue{1},
			Left:  avl.Pointer{Address: 3, Position: 4},
			Right: avl.Pointer{Address: 5, Position: 6},
			Valid: 1,
		}},
		Valid: 1,
	}
	candidate := fixedValueBlock{
		BlockAddress: 11,
		Leaf:         12,
		Data: NodeBlock[FixedValue]{Node: avl.Node[FixedValue]{
			Key:     avl.NewKey(fixedTestKey("candidat")),
			Value:   FixedValue{8},
			Balance: avl.BalanceRight,
			Left:    avl.Pointer{Address: 13, Position: 14},
			Right:   avl.Pointer{Address: 15, Position: 16},
			Valid:   1,
		}},
		Valid: 1,
	}
	if got := selectFixedValueBlockWords(current, candidate, 0); !reflect.DeepEqual(got, current) {
		t.Fatalf("condition 0 selected %+v, want current %+v", got, current)
	}
	if got := selectFixedValueBlockWords(current, candidate, 1); !reflect.DeepEqual(got, candidate) {
		t.Fatalf("condition 1 selected %+v, want candidate %+v", got, candidate)
	}
}

func fixedTestKey(value string) []byte {
	key := make([]byte, avl.KeySize)
	copy(key, value)
	return key
}

func hasPointers(value reflect.Type) bool {
	switch value.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Slice,
		reflect.String, reflect.Interface, reflect.Func, reflect.Chan:
		return true
	case reflect.Array:
		return hasPointers(value.Elem())
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if hasPointers(value.Field(index).Type) {
				return true
			}
		}
	}
	return false
}
