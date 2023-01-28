package di

import (
	"testing"

	"github.com/dozm/di/reflectx"
)

func TestContainerBuilder_Contains(t *testing.T) {
	b := Builder()
	AddTransient[int](b, func() int { return 1 })

	if !b.Contains(reflectx.TypeOf[int]()) {
		t.Error("assertion failed")
	}

	if b.Contains(reflectx.TypeOf[*int]()) {
		t.Error("assertion failed")
	}

	if b.Contains(reflectx.TypeOf[string]()) {
		t.Error("assertion failed")
	}
}

func TestContainerBuilder_Remove(t *testing.T) {
	b := Builder()
	AddTransient[int](b, func() int { return 1 })
	AddTransient[int](b, func() int { return 2 })
	AddTransient[string](b, func() string { return "a" })

	if !b.Contains(reflectx.TypeOf[int]()) {
		t.Error("assertion failed")
	}

	b.Remove(reflectx.TypeOf[int]())

	if b.Contains(reflectx.TypeOf[int]()) {
		t.Error("assertion failed")
	}

	if !b.Contains(reflectx.TypeOf[string]()) {
		t.Error("assertion failed")
	}

	c := b.Build()

	if _, err := TryGet[int](c); err == nil {
		t.Error("assertion failed")
	}

	if _, err := TryGet[string](c); err != nil {
		t.Error("assertion failed")
	}
}
