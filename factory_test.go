package di

import (
	"io"
	"strings"
	"sync/atomic"
	"testing"
)

type readWriter struct {
	Reader io.Reader
	Writer io.Writer
}

func (rw *readWriter) Read(p []byte) (n int, err error) {
	return rw.Reader.Read(p)
}

func (rw *readWriter) Write(p []byte) (n int, err error) {
	return rw.Writer.Write(p)
}

func readWriterFactory(c Container) io.ReadWriter {
	r, err := TryGet[io.Reader](c)
	if err != nil {
		panic(err)
	}
	w, err := TryGet[io.Writer](c)
	if err != nil {
		panic(err)
	}

	return &readWriter{
		Reader: r,
		Writer: w,
	}
}

func TestFactory_Basic(t *testing.T) {
	b := Builder()
	b.ConfigureOptions(func(o *Options) {
		// o.ValidateScopes = true
		o.ValidateOnBuild = true
	})

	s := "hello"

	b.Add(TransientFactory[io.ReadWriter](func(c Container) any { return readWriterFactory(c) }))
	b.Add(TransientFactory[io.Reader](func(c Container) any { return strings.NewReader(s) }))
	b.Add(ScopedFactory[io.Writer](func(c Container) any { return &strings.Builder{} }))

	c := b.Build()

	rw := Get[io.ReadWriter](c)

	v, ok := rw.(*readWriter)
	if !ok {
		t.Error("assertion failed")
	}

	_, ok = v.Reader.(*strings.Reader)
	if !ok {
		t.Error("assertion failed")
	}

	_, ok = v.Writer.(*strings.Builder)
	if !ok {
		t.Error("assertion failed")
	}

}

func TestFactory_Scoped(t *testing.T) {
	count := int32(0)
	b := Builder()
	b.ConfigureOptions(func(o *Options) {
		o.ValidateScopes = true
	})

	b.Add(ScopedFactory[int32](func(c Container) any { return atomic.AddInt32(&count, 1) }))

	c := b.Build()

	scope := Get[ScopeFactory](c).CreateScope()
	for i := 0; i < 10; i++ {
		v := Get[int32](scope.Container())
		if v != 1 {
			t.Error("assertion failed")
			return
		}
	}

	for i := int32(0); i < 10; i++ {
		scope := Get[ScopeFactory](c).CreateScope()
		v := Get[int32](scope.Container())
		if v != i+2 {
			t.Error("assertion failed")
			return
		}
	}
}
