package di

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/reflectx"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func TestContainer_ScopedService(t *testing.T) {
	value := int32(100)
	b := Builder()
	AddScoped[int32](b, func() int32 { return atomic.AddInt32(&value, 1) })

	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)

	numScopes, numValuesPerScope := 5, 5
	scopes := make([]Scope, numScopes)
	for i := 0; i < numScopes; i++ {
		scopes[i] = scopeFactory.CreateScope()
	}

	values := make([][]int32, len(scopes))
	for n := 0; n < numValuesPerScope; n++ {
		for i := 0; i < len(scopes); i++ {
			values[i] = append(values[i], Get[int32](scopes[i].Container()))
		}
	}

	for _, v := range values {
		if !allElementsEqual(v) {
			t.Error("values not equal in the same scope")
			break
		}
	}

	for i := 0; i < len(values); i++ {
		if i > 0 && values[i-1][0] == values[i][0] {
			t.Error("get the same values in different scopes")
			break
		}
	}
}

func allElementsEqual[T comparable](s []T) bool {
	for i := 0; i < len(s); i++ {
		if i > 0 && s[i-1] != s[i] {
			return false
		}
	}
	return true
}

func TestContainer_SliceAndDefaultValue(t *testing.T) {
	for i := 0; i <= 5; i++ {
		values := []int{1, 7, 5, 2, 8, 9}
		num := len(values)
		rand.Shuffle(num, func(i, j int) { values[i], values[j] = values[j], values[i] })
		b := Builder()

		for _, v := range values {
			v := v
			AddTransient[int](b, func() int { return v })
		}

		c := b.Build()
		results := Get[[]int](c)

		if len(results) != num {
			t.Error("unexpected length")
		}

		for i, v := range values {
			if results[i] != v {
				t.Error("unexpected value")
				break
			}
		}

		if v := Get[int](c); v != values[num-1] {
			t.Error("unexpected default value")
		}
	}

}

func TestContainer_ConstructorParameter(t *testing.T) {
	intValue := 99
	boolValue := true

	b := Builder()
	AddTransient[string](b, func(n int, b bool) string { return fmt.Sprintf("%v%v", n, b) })
	AddTransient[int](b, func(b bool) int { return intValue })
	AddTransient[bool](b, func() bool { return boolValue })

	c := b.Build()

	s := Get[string](c)

	if s != fmt.Sprintf("%v%v", intValue, boolValue) {
		t.Error("unexpected value")
	}
}

func TestContainer_ResolveSingletonConcurrently(t *testing.T) {
	rawValue := int32(100)
	value := rawValue
	expectedFinalValue := rawValue + 1

	b := Builder()
	AddSingleton[int32](b, func() int32 { return atomic.AddInt32(&value, 1) })

	c := b.Build()

	var wg sync.WaitGroup
	resolvedValues := make([]int32, 0)
	var mu sync.Mutex
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := Get[int32](c)
			mu.Lock()
			resolvedValues = append(resolvedValues, v)
			mu.Unlock()
		}()
	}

	wg.Wait()

	if value != expectedFinalValue {
		t.Errorf("expected %v actual %v", expectedFinalValue, value)
		return
	}

	for _, v := range resolvedValues {
		if v != expectedFinalValue {
			t.Errorf("expected %v actual %v", expectedFinalValue, v)
			break
		}
	}
}

type DisposableStruct struct {
	Value    int
	Disposed bool
}

func (d *DisposableStruct) Dispose() {
	d.Disposed = true
}

func TestContainer_DisposeSingletonWithConstructor(t *testing.T) {
	b := Builder()
	AddSingleton[*DisposableStruct](b, func() *DisposableStruct { return &DisposableStruct{} })

	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)

	scope := scopeFactory.CreateScope()
	obj := Get[*DisposableStruct](scope.Container())

	scope.Dispose()
	if obj.Disposed {
		t.Error("expect not be disposed")
		return
	}

	d, _ := c.(Disposable)
	d.Dispose()
	if !obj.Disposed {
		t.Error("expect disposed")
		return
	}
}

func TestContainer_NotDisposeConstant(t *testing.T) {
	b := Builder()
	AddInstance[*DisposableStruct](b, &DisposableStruct{})

	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)
	scope := scopeFactory.CreateScope()
	obj := Get[*DisposableStruct](scope.Container())

	scope.Dispose()
	if obj.Disposed {
		t.Error("expect not be disposed")
		return
	}

	d, _ := c.(Disposable)
	d.Dispose()

	if obj.Disposed {
		t.Error("expect not be disposed")
	}
}

func TestContainer_DisposableWithScope(t *testing.T) {
	b := Builder()
	AddScoped[*DisposableStruct](b, func() *DisposableStruct { return &DisposableStruct{} })

	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)

	scope1 := scopeFactory.CreateScope()
	obj1 := Get[*DisposableStruct](scope1.Container())

	scope1.Dispose()
	if !obj1.Disposed {
		t.Error("expect disposed")
		return
	}

	scope2 := scopeFactory.CreateScope()
	obj2 := Get[*DisposableStruct](scope2.Container())

	scope2.Dispose()
	if !obj2.Disposed {
		t.Error("expect disposed")
		return
	}
}

func TestContainer_ResolveScopedConcurrently(t *testing.T) {
	numScopes := 100
	concurrentPerScope := 100

	value := int32(0)
	b := Builder()
	AddScoped[*DisposableStruct](b,
		func() *DisposableStruct {
			return &DisposableStruct{
				Value: int(atomic.AddInt32(&value, 1)),
			}
		})

	c := b.Build()
	scopeFactory := Get[ScopeFactory](c)

	valuesScope := make([]int, 0)
	var muAppend sync.Mutex

	runNewScope := func() {
		scope := scopeFactory.CreateScope()
		var wg sync.WaitGroup
		var mu sync.Mutex
		resolvedValues := make([]*DisposableStruct, 0)

		for i := 0; i < concurrentPerScope; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v := Get[*DisposableStruct](scope.Container())
				mu.Lock()
				resolvedValues = append(resolvedValues, v)
				mu.Unlock()
			}()
		}

		wg.Wait()

		for _, v := range resolvedValues {
			if resolvedValues[0].Value != v.Value {
				t.Error("expect all of values is equal in the same scope")
			}
		}
		if resolvedValues[0].Disposed {
			t.Error("expect not be disposed")
		}
		scope.Dispose()
		if !resolvedValues[0].Disposed {
			t.Error("expect disposed")
		}

		muAppend.Lock()
		valueInThisScope := resolvedValues[0].Value
		duplicate := false
		for _, v := range valuesScope {
			if valueInThisScope == v {
				duplicate = true
				break
			}
		}
		valuesScope = append(valuesScope, valueInThisScope)
		muAppend.Unlock()
		if duplicate {
			t.Error("duplicate value")
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < numScopes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runNewScope()
		}()
	}

	wg.Wait()

	if int(value) != numScopes {
		t.Error("expect the final value is equal to the number of scopes")
	}
}

func TestContainer_ResoleScopedServiceFromRoot(t *testing.T) {
	b := Builder()
	b.ConfigureOptions(func(opts *Options) {
		opts.ValidateScopes = true
	})

	AddScoped[int](b, func() int { return 1 })
	AddTransient[string](b, func(i int) string { return "" })

	c := b.Build()

	_, err := TryGet[int](c)

	if _, ok := err.(*errorx.ScopedServiceFromRootError); !ok {
		t.Errorf("expect an error of type '%v'", reflectx.TypeOf[errorx.ScopedServiceFromRootError]())
	}

	_, err2 := TryGet[string](c)
	if _, ok := err2.(*errorx.ScopedServiceFromRootError); !ok {
		t.Errorf("expect an error of type '%v'", reflectx.TypeOf[errorx.ScopedServiceFromRootError]())
	}

	scope := Get[ScopeFactory](c).CreateScope()

	if _, err := TryGet[int](scope.Container()); err != nil {
		t.Errorf("expect no error")
	}

	if _, err := TryGet[string](scope.Container()); err != nil {
		t.Errorf("expect no error")
	}
}

func TestContainer_SliceElementWithDifferentLifetime(t *testing.T) {
	intValue := int32(0)
	b := Builder()
	AddSingleton[int](b, func() int { return int(atomic.AddInt32(&intValue, 1)) })
	AddScoped[int](b, func() int { return int(atomic.AddInt32(&intValue, 1)) })
	AddTransient[int](b, func() int { return int(atomic.AddInt32(&intValue, 1)) })

	c := b.Build()

	scope1 := Get[ScopeFactory](c).CreateScope()
	sliceScope1_1 := Get[[]int](scope1.Container())
	sliceScope1_2 := Get[[]int](scope1.Container())

	if sliceScope1_1[0] != sliceScope1_2[0] ||
		sliceScope1_1[1] != sliceScope1_2[1] ||
		sliceScope1_1[2] == sliceScope1_2[2] {

		t.Error("assertion failed")
	}

	scope2 := Get[ScopeFactory](c).CreateScope()
	sliceScope2_1 := Get[[]int](scope2.Container())

	if sliceScope2_1[0] != sliceScope1_1[0] ||
		sliceScope2_1[1] == sliceScope1_1[1] ||
		sliceScope2_1[2] == sliceScope1_1[2] {

		t.Error("assertion failed")
	}
}

func TestContainer_IsService(t *testing.T) {
	b := Builder()
	AddTransient[int](b, func() int { return 1 })
	c := b.Build()

	isService := Get[IsService](c)

	if !isService.IsService(reflectx.TypeOf[int]()) {
		t.Error("assertion failed")
	}

	if isService.IsService(reflectx.TypeOf[string]()) {
		t.Error("assertion failed")
	}
}

func TestContainer_ValidateOnBuild(t *testing.T) {
	b := Builder()
	b.ConfigureOptions(func(o *Options) {
		o.ValidateOnBuild = false
	})

	AddTransient[int](b, func(s string) int { return 1 })
	AddTransient[string](b, func(i int) string { return "" })

	var err interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = r
			}
		}()

		_ = b.Build()
	}()

	if err != nil {
		t.Error("assertion failed")
	}

	b.ConfigureOptions(func(o *Options) {
		o.ValidateOnBuild = true
	})

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = r
			}
		}()

		_ = b.Build()
	}()

	if err == nil {
		t.Error("assertion failed")
	}
}

func TestInvoke(t *testing.T) {
	b := Builder()
	AddInstance[int](b, 100)
	AddInstance[int16](b, int16(0))
	AddInstance[int32](b, int32(3))
	c := b.Build()

	divide := func(a int, b int) (float32, error) {
		if b == 0 {
			return 0, errors.New("divide by zero")
		}
		return float32(a) / float32(b), nil
	}

	results, err := Invoke(c,
		func(a int, b int32) (float32, error) {
			return divide(a, int(b))
		})

	if err != nil {
		t.Error("assertion failed")
	}

	if len(results) != 2 {
		t.Error("assertion failed")
		return
	}

	if _, ok := results[0].(float32); !ok {
		t.Error("assertion failed")
		return
	}

	if results[1] != nil {
		t.Error("assertion failed")
		return
	}

	results, _ = Invoke(c,
		func(a int, b int16) (float32, error) {
			return divide(a, int(b))
		})

	if _, ok := results[1].(error); !ok {
		t.Error("assertion failed")
		return
	}

	_, err = Invoke(c,
		func(a int, b int64) (float32, error) {
			return divide(a, int(b))
		})

	if _, ok := err.(*errorx.ServiceNotFound); !ok {
		t.Error("assertion failed")
	}
}
