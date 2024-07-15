package ptr_test

import (
	"testing"

	"github.com/krateoplatformops/oasgen-provider/internal/crdgen/internal/ptr"
)

func TestRef(t *testing.T) {
	type T int

	val := T(0)
	pointer := ptr.To(val)
	if *pointer != val {
		t.Errorf("expected %d, got %d", val, *pointer)
	}

	val = T(1)
	pointer = ptr.To(val)
	if *pointer != val {
		t.Errorf("expected %d, got %d", val, *pointer)
	}
}

func TestDeref(t *testing.T) {
	type T int

	var val, def T = 1, 0

	out := ptr.Deref(&val, def)
	if out != val {
		t.Errorf("expected %d, got %d", val, out)
	}

	out = ptr.Deref(nil, def)
	if out != def {
		t.Errorf("expected %d, got %d", def, out)
	}
}

func TestEqual(t *testing.T) {
	type T int

	if !ptr.Equal[T](nil, nil) {
		t.Errorf("expected true (nil == nil)")
	}
	if !ptr.Equal(ptr.To(T(123)), ptr.To(T(123))) {
		t.Errorf("expected true (val == val)")
	}
	if ptr.Equal(nil, ptr.To(T(123))) {
		t.Errorf("expected false (nil != val)")
	}
	if ptr.Equal(ptr.To(T(123)), nil) {
		t.Errorf("expected false (val != nil)")
	}
	if ptr.Equal(ptr.To(T(123)), ptr.To(T(456))) {
		t.Errorf("expected false (val != val)")
	}
}
