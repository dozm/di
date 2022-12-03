package di

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/syncx"
)

type validatorState struct {
	Singleton CallSite
}

type CallSiteValidator struct {
	scopedServices *syncx.Map[reflect.Type, reflect.Type]
}

func (v *CallSiteValidator) ValidateCallSite(callSite CallSite) error {
	scoped, err := v.visitCallSite(callSite, validatorState{})
	if err != nil {
		return err
	}

	if scoped != nil {
		v.scopedServices.Store(callSite.ServiceType(), scoped)
	}

	return nil
}

func (v *CallSiteValidator) ValidateResolution(serviceType reflect.Type, scope Scope, rootScope Scope) (err error) {
	if scope == rootScope {
		scopedService, ok := v.scopedServices.Load(serviceType)
		if !ok {
			return
		}
		if serviceType == scopedService {
			return &errorx.ScopedServiceFromRootError{
				Message: fmt.Sprintf("cannot resolve scoped service '%v' from root scope", serviceType)}
		}

		return &errorx.ScopedServiceFromRootError{
			Message: fmt.Sprintf("cannot resolve '%v' from root scope because it requires scoped service '%v'", serviceType, scopedService),
		}
	}
	return
}

func (r *CallSiteValidator) visitCallSite(callSite CallSite, state validatorState) (reflect.Type, error) {
	switch callSite.Cache().Location {
	case CacheLocation_Root:
		return r.visitRootCache(callSite, state)
	case CacheLocation_Scope:
		return r.visitScopeCache(callSite, state)
	case CacheLocation_Dispose:
		return r.visitDisposeCache(callSite, state)
	case CacheLocation_None:
		return r.visitNoCache(callSite, state)
	default:
		return nil, errors.New("unknow cache location")
	}
}

func (r *CallSiteValidator) visitCallSiteMain(callSite CallSite, state validatorState) (reflect.Type, error) {
	switch callSite.Kind() {
	case CallSiteKind_Slice:
		return r.visitSlice(callSite.(*SliceCallSite), state)
	case CallSiteKind_Constructor:
		return r.visitConstructor(callSite.(*ConstructorCallSite), state)
	case CallSiteKind_Constant:
		return r.visitConstant(callSite.(*ConstantCallSite), state)
	case CallSiteKind_Container:
		return r.visitContainer(callSite.(*ContainerCallSite), state)
	default:
		return nil, errors.New("unknow call site kind")
	}
}

func (v *CallSiteValidator) visitConstructor(callSite *ConstructorCallSite, state validatorState) (reflect.Type, error) {
	var result reflect.Type
	for _, cs := range callSite.Parameters {
		scoped, err := v.visitCallSite(cs, state)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = scoped
		}
	}

	return result, nil
}

func (v *CallSiteValidator) visitSlice(callSite *SliceCallSite, state validatorState) (reflect.Type, error) {
	var result reflect.Type
	for _, cs := range callSite.CallSites {
		scoped, err := v.visitCallSite(cs, state)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = scoped
		}
	}
	return result, nil
}

func (v *CallSiteValidator) visitRootCache(singletonCallSite CallSite, state validatorState) (reflect.Type, error) {
	state.Singleton = singletonCallSite
	return v.visitCallSiteMain(singletonCallSite, state)
}

func (v *CallSiteValidator) visitScopeCache(scopedCallSite CallSite, state validatorState) (reflect.Type, error) {
	if scopedCallSite.ServiceType() == ScopeFactoryType {
		return nil, nil
	}

	if state.Singleton != nil {
		return nil, fmt.Errorf("cannot consume scoped service '%v' from singleton '%v'",
			scopedCallSite.ServiceType(),
			state.Singleton.ServiceType())
	}
	_, err := v.visitCallSiteMain(scopedCallSite, state)
	if err != nil {
		return nil, err
	}

	return scopedCallSite.ServiceType(), nil
}

func (v *CallSiteValidator) visitDisposeCache(callSite CallSite, state validatorState) (reflect.Type, error) {
	return v.visitCallSiteMain(callSite, state)
}

func (v *CallSiteValidator) visitNoCache(callSite CallSite, state validatorState) (reflect.Type, error) {
	return v.visitCallSiteMain(callSite, state)
}

func (v *CallSiteValidator) visitConstant(callSite *ConstantCallSite, state validatorState) (reflect.Type, error) {
	return nil, nil
}

func (v *CallSiteValidator) visitContainer(callSite *ContainerCallSite, state validatorState) (reflect.Type, error) {
	return nil, nil
}

func newCallSiteValidator() *CallSiteValidator {
	return &CallSiteValidator{scopedServices: syncx.NewMap[reflect.Type, reflect.Type]()}
}
