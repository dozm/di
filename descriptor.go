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
	ServiceType               reflect.Type
	Lifetime                  Lifetime
	Ctor                      *ConstructorInfo
	Instance                  any
	Factory                   func(Container) any
	ImplementedInterfaceTypes []reflect.Type
	LookupKeys                []string
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

// validateServiceType validates the service type.
// panics if implementedInterfaceTypes is passed then the serviceType MUST be a struct.
// panics if implementedInterfaceTypes must be interfaces and the serviceType must implement them.
func validateServiceType(serviceType reflect.Type, implementedInterfaceTypes ...reflect.Type) {
	if len(implementedInterfaceTypes) > 0 {
		kind := serviceType.Kind()
		// if serviceType is a pointer, get the element type
		if kind != reflect.Ptr {
			panic(fmt.Errorf("if implementedInterfaceTypes is passed then the serviceType MUST be a struct ptr.  i.e. *MyStruct"))
		}
		serviceTypeElem := serviceType.Elem()

		if serviceTypeElem.Kind() != reflect.Struct {
			panic(fmt.Errorf("if implementedInterfaceTypes is passed then the serviceType MUST be a struct ptr.  i.e. *MyStruct"))
		}
		for _, t := range implementedInterfaceTypes {
			kind := t.Kind()
			// if t is a pointer, get the element type
			if kind == reflect.Ptr {
				t = t.Elem()
			}
			if t.Kind() != reflect.Interface {
				panic(fmt.Errorf("implementedInterfaceTypes must be interfaces. i.e. reflect.TypeOf((*ITime)(nil))"))
			}
			if !serviceType.Implements(t) {
				panic(fmt.Errorf("the serviceType must implement the interface '%v'", t))
			}
		}
	}
}
func NewInstanceDescriptor(serviceType reflect.Type, instance any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	if err := instanceAssignable(instance, serviceType); err != nil {
		panic(err)
	}
	validateServiceType(serviceType, implementedInterfaceTypes...)
	var implementedInterfaceTypesElem []reflect.Type
	for _, t := range implementedInterfaceTypes {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		implementedInterfaceTypesElem = append(implementedInterfaceTypesElem, t)
	}
	return &Descriptor{
		ServiceType:               serviceType,
		Lifetime:                  Lifetime_Singleton,
		Instance:                  instance,
		ImplementedInterfaceTypes: implementedInterfaceTypesElem,
	}
}

func NewConstructorDescriptor(serviceType reflect.Type, lifetime Lifetime, ctor any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	ci := newConstructorInfo(ctor)
	err := checkConstructor(ci, serviceType)

	if err != nil {
		panic(err)
	}

	validateServiceType(serviceType, implementedInterfaceTypes...)
	var implementedInterfaceTypesElem []reflect.Type
	for _, t := range implementedInterfaceTypes {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		implementedInterfaceTypesElem = append(implementedInterfaceTypesElem, t)
	}
	return &Descriptor{
		ServiceType:               serviceType,
		Lifetime:                  lifetime,
		Ctor:                      ci,
		ImplementedInterfaceTypes: implementedInterfaceTypesElem,
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

func hashTypeAndString(t reflect.Type, s string) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return fmt.Sprintf("%s-%s", t.Name(), s)
}
