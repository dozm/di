package di

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dozm/di/syncx"
)

type resolverLock byte

const (
	resolverLock_Scope resolverLock = 1
	resolverLock_Root  resolverLock = 2
)

var CallSiteResolverInstance *CallSiteResolver = newCallSiteResolver()

type resolverContext struct {
	Scope         *ContainerEngineScope
	AcquiredLocks resolverLock
}

type CallSiteResolver struct {
	callSiteLockers *syncx.LockMap
}

func (r *CallSiteResolver) Resolve(callSite CallSite, scope *ContainerEngineScope) (any, error) {
	if scope.IsRootScope {
		if cached := callSite.Value(); cached != nil {
			return cached, nil
		}
	}

	return r.visitCallSite(callSite, resolverContext{Scope: scope})
}

func (r *CallSiteResolver) visitCallSite(callSite CallSite, ctx resolverContext) (any, error) {
	switch callSite.Cache().Location {
	case CacheLocation_Root:
		return r.visitRootCache(callSite, ctx)
	case CacheLocation_Scope:
		return r.visitScopeCache(callSite, ctx)
	case CacheLocation_Dispose:
		return r.visitDisposeCache(callSite, ctx)
	case CacheLocation_None:
		return r.visitNoCache(callSite, ctx)
	default:
		return nil, errors.New("unknow cache location")
	}
}

func (r *CallSiteResolver) visitCallSiteMain(callSite CallSite, ctx resolverContext) (any, error) {
	switch callSite.Kind() {
	case CallSiteKind_Slice:
		return r.visitSlice(callSite.(*SliceCallSite), ctx)
	case CallSiteKind_Constructor:
		return r.visitConstructor(callSite.(*ConstructorCallSite), ctx)
	case CallSiteKind_Constant:
		return r.visitConstant(callSite.(*ConstantCallSite), ctx)
	case CallSiteKind_Container:
		return r.visitContainer(callSite.(*ContainerCallSite), ctx)
	default:
		return nil, errors.New("unknow call site kind")
	}
}

func (r *CallSiteResolver) visitNoCache(callSite CallSite, ctx resolverContext) (any, error) {
	return r.visitCallSiteMain(callSite, ctx)
}

func (r *CallSiteResolver) visitDisposeCache(transientCallSite CallSite, ctx resolverContext) (any, error) {
	v, err := r.visitCallSiteMain(transientCallSite, ctx)
	if err != nil {
		return nil, err
	}

	_, err = ctx.Scope.CaptureDisposable(v)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (r *CallSiteResolver) visitConstructor(callSite *ConstructorCallSite, ctx resolverContext) (any, error) {
	numParams := len(callSite.Parameters)
	inValues := make([]reflect.Value, numParams)
	if numParams > 0 {
		var v any
		var err error
		for i, p := range callSite.Parameters {
			if v, err = r.visitCallSite(p, ctx); err != nil {
				return nil, err
			}
			inValues[i] = reflect.ValueOf(v)
		}
	}

	outValues := callSite.Ctor.Call(inValues)

	numOut := len(outValues)
	if numOut == 1 {
		return outValues[0].Interface(), nil
	} else if numOut == 2 {
		if outValues[1].IsZero() {
			return outValues[0].Interface(), nil
		}
		if err, ok := outValues[1].Interface().(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("the type of the second out parameter is not error")
	} else {
		return nil, fmt.Errorf("unexpected output parameters")
	}
}

func (r *CallSiteResolver) visitRootCache(callSite CallSite, ctx resolverContext) (any, error) {
	if value := callSite.Value(); value != nil {
		return value, nil
	}

	rootScope := ctx.Scope.RootContainer.Root

	callSiteLocker := r.callSiteLockers.LoadOrCreate(callSite)
	callSiteLocker.Lock()
	defer callSiteLocker.Unlock()

	if value := callSite.Value(); value != nil {
		return value, nil
	}

	resolved, err := r.visitCallSiteMain(callSite, resolverContext{
		Scope:         rootScope,
		AcquiredLocks: ctx.AcquiredLocks | resolverLock_Root,
	})

	if err != nil {
		return nil, err
	}

	_, err = rootScope.CaptureDisposable(resolved)
	if err != nil {
		return nil, err
	}
	callSite.SetValue(resolved)
	return resolved, nil
}

func (r *CallSiteResolver) visitScopeCache(callSite CallSite, ctx resolverContext) (any, error) {
	scope := ctx.Scope
	if scope.IsRootScope {
		return r.visitRootCache(callSite, ctx)
	}

	resolvedServices := scope.ResolvedServices
	cacheKey := callSite.Cache().Key

	if (ctx.AcquiredLocks & resolverLock_Scope) == 0 {
		scope.Locker.Lock()
		defer scope.Locker.Unlock()
	}

	if resolved, ok := resolvedServices[cacheKey]; ok {
		return resolved, nil
	}

	resolved, err := r.visitCallSiteMain(callSite, resolverContext{
		Scope:         scope,
		AcquiredLocks: ctx.AcquiredLocks | resolverLock_Scope,
	})
	if err != nil {
		return nil, err
	}

	if _, err = scope.CaptureDisposableWithoutLock(resolved); err != nil {
		return nil, err
	}

	resolvedServices[cacheKey] = resolved
	return resolved, nil
}

func (r *CallSiteResolver) visitConstant(callSite *ConstantCallSite, ctx resolverContext) (any, error) {
	return callSite.DefaultValue(), nil
}

func (r *CallSiteResolver) visitContainer(callSite *ContainerCallSite, ctx resolverContext) (any, error) {
	return ctx.Scope, nil
}

func (r *CallSiteResolver) visitSlice(callSite *SliceCallSite, ctx resolverContext) (any, error) {
	size := len(callSite.CallSites)
	s := reflect.MakeSlice(callSite.ServiceType(), size, size)

	var v any
	var err error
	for i, cs := range callSite.CallSites {
		v, err = r.visitCallSite(cs, ctx)
		if err != nil {
			return nil, err
		}
		s.Index(i).Set(reflect.ValueOf(v))
	}

	return s.Interface(), nil
}

func newCallSiteResolver() *CallSiteResolver {
	return &CallSiteResolver{
		callSiteLockers: &syncx.LockMap{},
	}
}
