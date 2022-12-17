package di

import (
	"fmt"
	"reflect"

	"github.com/dozm/di/reflectx"
)

type Lifetime byte

const (
	Lifetime_Singleton Lifetime = iota
	Lifetime_Scoped
	Lifetime_Transient
)

type Factory func(Container) any

type ConstructorInfo struct {
	FuncType  reflect.Type
	FuncValue reflect.Value
	// input parameter types
	In []reflect.Type
	// output parameter types
	Out []reflect.Type
}

func (c *ConstructorInfo) Call(params []reflect.Value) []reflect.Value {
	return c.FuncValue.Call(params)
}

func newConstructorInfo(ctor any) *ConstructorInfo {
	ft := reflect.TypeOf(ctor)
	return &ConstructorInfo{
		FuncValue: reflect.ValueOf(ctor),
		FuncType:  ft,
		In:        reflectx.GetInParameters(ft),
		Out:       reflectx.GetOutParameters(ft),
	}

}

// service descriptor
type Descriptor struct {
	ServiceType reflect.Type
	Lifetime    Lifetime
	Ctor        *ConstructorInfo
	Instance    any
	Factory     func(Container) any
}

func (d *Descriptor) String() string {
	s := fmt.Sprintf("ServiceType: %v Lifetime: %v ", d.ServiceType, d.Lifetime)

	if d.Ctor != nil {
		s += fmt.Sprintf("Constructor: %v", d.Ctor.FuncType)
	} else {
		s += fmt.Sprintf("Instance: %v", d.Instance)
	}

	return s
}

func NewInstanceDescriptor(serviceType reflect.Type, instance any) *Descriptor {
	if err := instanceAssignable(instance, serviceType); err != nil {
		panic(err)
	}

	return &Descriptor{
		ServiceType: serviceType,
		Lifetime:    Lifetime_Singleton,
		Instance:    instance,
	}
}

func NewConstructorDescriptor(serviceType reflect.Type, lifetime Lifetime, ctor any) *Descriptor {
	ci := newConstructorInfo(ctor)
	err := checkConstructor(ci, serviceType)

	if err != nil {
		panic(err)
	}

	return &Descriptor{
		ServiceType: serviceType,
		Lifetime:    lifetime,
		Ctor:        ci,
	}
}

func checkConstructor(ctor *ConstructorInfo, serviceType reflect.Type) (err error) {
	if ctor.FuncType.Kind() != reflect.Func {
		return fmt.Errorf("the constructor of the service '%v' is not a function", serviceType)
	}

	out := ctor.Out
	numOut := len(out)
	if (numOut == 0 || numOut > 2) ||
		!out[0].AssignableTo(serviceType) ||
		(numOut == 2 && !reflectx.IsErrorType(out[1])) {
		return fmt.Errorf("the constructor must returns a '%v' and an optional error", serviceType)
	}

	return
}

func instanceAssignable(instance any, to reflect.Type) (err error) {
	if t := reflect.TypeOf(instance); !t.AssignableTo(to) {
		err = fmt.Errorf("the instance of type '%v' can not assignable to type '%v'", t, to)
	}
	return
}

func NewFactoryDescriptor(serviceType reflect.Type, lifetime Lifetime, factory Factory) *Descriptor {
	return &Descriptor{
		ServiceType: serviceType,
		Lifetime:    lifetime,
		Factory:     factory,
	}
}
