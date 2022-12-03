package di

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/reflectx"
)

type Iface1 interface{ I1_F() }
type Iface2 interface{ I2_F() }

type A struct{}
type B struct{}
type C struct{}
type D struct{}
type E struct{}
type F struct{}
type G struct{}

func (d *D) I1_F() {}
func (e *E) I1_F() {}

func (d *D) I2_F() {}
func (e *E) I2_F() {}
func (f *F) I2_F() {}
func (f *G) I2_F() {}

var Iface2Descriptors = func() []*Descriptor {
	newD := func() *D { return &D{} }
	newE := func() *E { return &E{} }
	newF := func() *F { return &F{} }
	newG := func() *G { return &G{} }

	iface2Type := reflectx.TypeOf[Iface2]()

	return []*Descriptor{
		NewConstructorDescriptor(iface2Type, Lifetime_Transient, newD),
		NewConstructorDescriptor(iface2Type, Lifetime_Transient, newE),
		NewConstructorDescriptor(iface2Type, Lifetime_Transient, newF),
		NewConstructorDescriptor(iface2Type, Lifetime_Transient, newG),
	}
}()

func TestCallSiteFactory_ServiceNotRegistered(t *testing.T) {
	newA := func() *A { return nil }

	descriptors := []*Descriptor{
		NewConstructorDescriptor(reflectx.TypeOf[*A](), Lifetime_Transient, newA),
	}
	callSiteFactory := newCallSiteFactory(descriptors)

	if cs, _ := callSiteFactory.GetCallSite(reflectx.TypeOf[*A](), newCallSiteChain()); cs == nil {
		t.Error("assertion failed")
	}

	if _, err := callSiteFactory.GetCallSite(reflectx.TypeOf[A](), newCallSiteChain()); err == nil {
		t.Error("assertion failed")
	}

	if _, err := callSiteFactory.GetCallSite(reflectx.TypeOf[B](), newCallSiteChain()); err == nil {
		t.Error("assertion failed")
	}

}

func TestCallSiteFactory_CircularDependency(t *testing.T) {
	newA := func(b B) A { return A{} }
	newB := func(c C) B { return B{} }
	newC := func(d D) C { return C{} }
	newD := func(b B, e E) D { return D{} }
	newE := func() E { return E{} }

	descriptors := []*Descriptor{
		NewConstructorDescriptor(reflect.TypeOf(A{}), Lifetime_Transient, newA),
		NewConstructorDescriptor(reflect.TypeOf(B{}), Lifetime_Transient, newB),
		NewConstructorDescriptor(reflect.TypeOf(C{}), Lifetime_Transient, newC),
		NewConstructorDescriptor(reflect.TypeOf(D{}), Lifetime_Transient, newD),
		NewConstructorDescriptor(reflect.TypeOf(E{}), Lifetime_Transient, newE),
	}

	callSiteFactory := newCallSiteFactory(descriptors)
	_ = callSiteFactory

	_, err := callSiteFactory.GetCallSite(reflect.TypeOf(A{}), newCallSiteChain())
	if _, ok := err.(*errorx.CircularDependencyError); !ok {
		t.Error("assertion failed")
	}

}

func TestCallSiteFactory_ImplicitSlice(t *testing.T) {
	numIface2Descriptor := len(Iface2Descriptors)

	descriptors := append([]*Descriptor{}, Iface2Descriptors...)
	callSiteFactory := newCallSiteFactory(descriptors)

	iface2SliceType := reflectx.TypeOf[[]Iface2]()
	callSite, err := callSiteFactory.GetCallSite(iface2SliceType, newCallSiteChain())
	if err != nil {
		t.Error(err)
		return
	}

	cs, ok := callSite.(*SliceCallSite)
	if !ok {
		t.Errorf("expect %v, actual: %v", reflectx.TypeOf[*SliceCallSite](), reflect.TypeOf(callSite))
	}

	numCallSite := len(cs.CallSites)
	if numCallSite != numIface2Descriptor {
		t.Errorf("expect %v, actual: %v", numIface2Descriptor, numCallSite)
	}
}

func TestCallSiteFactory_ExactSlice(t *testing.T) {
	descriptors := append([]*Descriptor{}, Iface2Descriptors...)
	iface2SliceType := reflectx.TypeOf[[]Iface2]()

	iface2slice := []Iface2{&F{}, &F{}}
	newIface2Slice := func() []Iface2 { return iface2slice }
	descriptors = append(
		descriptors,
		NewConstructorDescriptor(iface2SliceType, Lifetime_Transient, newIface2Slice),
	)
	callSiteFactory := newCallSiteFactory(descriptors)

	callSite, err := callSiteFactory.GetCallSite(iface2SliceType, newCallSiteChain())
	if err != nil {
		t.Error(err)
		return
	}

	_, ok := callSite.(*ConstructorCallSite)
	if !ok {
		t.Errorf("expect %v, actual: %v", reflectx.TypeOf[*ConstructorCallSite](), reflect.TypeOf(callSite))
	}
}

func TestCallSiteFactory_EmptySlice(t *testing.T) {
	descriptors := append([]*Descriptor{}, Iface2Descriptors...)
	callSiteFactory := newCallSiteFactory(descriptors)

	callSite, err := callSiteFactory.GetCallSite(reflectx.TypeOf[[]Iface1](), newCallSiteChain())
	if err != nil {
		t.Error(err)
		return
	}

	sliceCallSite, ok := callSite.(*SliceCallSite)
	if !ok {
		t.Errorf("expect %v, actual: %v", reflectx.TypeOf[*SliceCallSite](), reflect.TypeOf(callSite))
	}

	if len(sliceCallSite.CallSites) != 0 {
		t.Errorf("expect an empty slice")
	}
}

func Shuffle[T any](s []T) {
	rand.Seed(time.Now().UTC().UnixNano())
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

func TestCallSiteFactory_Last(t *testing.T) {
	ctors := []func() int{}
	descriptors := []*Descriptor{}
	n := 10
	for i := 0; i < n; i++ {
		v := i
		ctors = append(ctors, func() int { return v })
	}
	Shuffle(ctors)
	for _, ctor := range ctors {
		descriptors = append(descriptors,
			NewConstructorDescriptor(reflectx.TypeOf[int](), Lifetime_Transient, ctor))
	}

	callSiteFactory := newCallSiteFactory(descriptors)

	callSite, err := callSiteFactory.GetCallSite(reflectx.TypeOf[int](), newCallSiteChain())
	if err != nil {
		t.Error(err)
		return
	}

	ccs, ok := callSite.(*ConstructorCallSite)
	if !ok {
		t.Errorf("expect %v, actual: %v", reflectx.TypeOf[*SliceCallSite](), reflect.TypeOf(callSite))
	}

	ctor, _ := ccs.Ctor.FuncValue.Interface().(func() int)
	if ctor() != ctors[n-1]() {
		t.Errorf("expect %v, actual: %v", ctors[n-1](), ctor())
	}
}
