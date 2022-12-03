package di

import (
	"errors"
	"reflect"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/reflectx"
)

type Container interface {
	Get(reflect.Type) (any, error)
}

type Scope interface {
	Container() Container
	Dispose()
}

type ScopeFactory interface {
	CreateScope() Scope
}

// Optional service used to determine if the specified type is available from the Container.
type IsService interface {
	IsService(serviceType reflect.Type) bool
}

type Disposable interface {
	Dispose()
}

// Get service of the type T from the container c
func Get[T any](c Container) T {
	result, err := TryGet[T](c)
	if err != nil {
		panic(err)
	}
	return result
}

func TryGet[T any](c Container) (result T, err error) {
	t := reflectx.TypeOf[T]()
	v, err := c.Get(t)
	if err != nil {
		return
	}

	result, ok := v.(T)
	if !ok {
		err = &errorx.TypeIncompatibilityError{To: t, From: reflect.TypeOf(v)}
		return
	}

	return
}

// Invoke the function fn.
// the input paramenters of the fn function will be resolved from the Container c.
func Invoke(c Container, fn any) (fnReturn []any, err error) {
	vfn := reflect.ValueOf(fn)
	if vfn.Kind() != reflect.Func {
		err = errors.New("fn is not a function")
		return
	}

	inputTypes := reflectx.GetInParameters(vfn.Type())

	inputs := make([]reflect.Value, len(inputTypes))
	for i, t := range inputTypes {
		v, e := c.Get(t)
		if e != nil {
			err = e
			return
		}

		inputs[i] = reflect.ValueOf(v)
	}

	ouputs := vfn.Call(inputs)
	numOutputs := len(ouputs)
	if numOutputs > 0 {
		fnReturn = make([]any, numOutputs)
		for i, v := range ouputs {
			fnReturn[i] = v.Interface()
		}
	}

	return
}
