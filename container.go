package di

import (
	"fmt"
	"reflect"

	"github.com/dozm/di/reflectx"
	"github.com/dozm/di/syncx"
)

var ContainerType = reflectx.TypeOf[Container]()
var ContainerImplType = reflectx.TypeOf[container]()
var ScopeFactoryType = reflectx.TypeOf[ScopeFactory]()
var IsServiceType = reflectx.TypeOf[IsService]()

// Container options.
type Options struct {
	ValidateScopes  bool
	ValidateOnBuild bool
}

// Get default container options.
func DefaultOptions() Options {
	return Options{}
}

// Container implementation
type container struct {
	Root              *ContainerEngineScope
	CallSiteFactory   *CallSiteFactory
	engine            ContainerEngine
	realizedServices  *syncx.Map[reflect.Type, ServiceAccessor]
	disposed          bool
	callSiteValidator *CallSiteValidator
}

func (c *container) Get(serviceType reflect.Type) (any, error) {
	return c.GetWithScope(serviceType, c.Root)
}

func (c *container) CreateScope() Scope {
	if c.disposed {
		panic(fmt.Errorf("%v disposed", reflect.TypeOf(c).Elem()))
	}

	return newEngineScope(c, false)
}

func (c *container) GetWithScope(serviceType reflect.Type, scope *ContainerEngineScope) (result any, err error) {
	if c.disposed {
		err = fmt.Errorf("%v disposed", reflect.TypeOf(c).Elem())
		return
	}

	defer func() {
		if p := recover(); p != nil {
			if e, ok := p.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", p)
			}
		}
	}()

	accessor, ok := c.realizedServices.Load(serviceType)
	if !ok {
		accessor, err = c.createServiceAccessor(serviceType)
		if err != nil {
			return
		} else {
			accessor, _ = c.realizedServices.LoadOrStore(serviceType, accessor)
		}

	}

	if c.callSiteValidator != nil {
		err := c.callSiteValidator.ValidateResolution(serviceType, scope, c.Root)
		if err != nil {
			return nil, err
		}
	}

	return accessor(scope)
}

func (c *container) validateService(d *Descriptor) error {
	callSite, err := c.CallSiteFactory.GetCallSiteByDescriptor(d, newCallSiteChain())
	if err != nil {
		return err
	}
	if c.callSiteValidator != nil {
		return c.callSiteValidator.ValidateCallSite(callSite)
	}
	return nil
}

func (c *container) Dispose() {
	c.disposed = true
	c.Root.Dispose()
}

func (c *container) IsDisposed() bool {
	return c.disposed
}

func (c *container) createEngine() ContainerEngine {
	return newContainerEngine(c)
}

func (c *container) createServiceAccessor(serviceType reflect.Type) (ServiceAccessor, error) {
	callSite, err := c.CallSiteFactory.GetCallSite(serviceType, newCallSiteChain())
	if err != nil {
		return nil, err
	}

	if c.callSiteValidator != nil {
		if err := c.callSiteValidator.ValidateCallSite(callSite); err != nil {
			return nil, err
		}
	}

	if callSite.Cache().Location == CacheLocation_Root {
		value, err := CallSiteResolverInstance.Resolve(callSite, c.Root)
		if err != nil {
			return nil, err
		}
		return func(scope *ContainerEngineScope) (any, error) { return value, nil }, nil
	}

	return c.engine.RealizeService(callSite)
}

func (c *container) ReplaceServiceAccessor(callSite CallSite, accessor ServiceAccessor) {
	c.realizedServices.Store(callSite.ServiceType(), accessor)
}
