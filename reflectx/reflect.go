package reflectx

import (
	"fmt"
	"reflect"
	"runtime"
)

func GetOutParameters(funcType reflect.Type) []reflect.Type {
	if funcType.Kind() != reflect.Func {
		panic(fmt.Errorf("the kind of type '%v' is not function", funcType))
	}
	n := funcType.NumOut()
	paramTypes := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		paramTypes[i] = funcType.Out(i)
	}
	return paramTypes
}

func GetInParameters(funcType reflect.Type) []reflect.Type {
	if funcType.Kind() != reflect.Func {
		panic(fmt.Errorf("the kind of type '%v' is not function", funcType))
	}
	n := funcType.NumIn()
	paramTypes := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		paramTypes[i] = funcType.In(i)
	}
	return paramTypes
}

func IsErrorType(t reflect.Type) bool {
	et := reflect.TypeOf((*error)(nil)).Elem()
	return t.AssignableTo(et)
}

func GetFuncName(f any) string {
	rv := reflect.ValueOf(f)
	if rv.Kind() != reflect.Func {
		panic("the argument is not a function")
	}
	return runtime.FuncForPC(rv.Pointer()).Name()
}

func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
