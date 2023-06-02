package di

import (
	"reflect"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/reflectx"
	"github.com/dozm/di/syncx"
)

type ContainerBuilder interface {
	Add(...*Descriptor)
	// Remove all the descriptors that the service type is t.
	Remove(t reflect.Type)
	Contains(t reflect.Type) bool
	Build() Container
	ConfigureOptions(func(*Options))
}

type containerBuilder struct {
	descriptors          []*Descriptor
	optionsConfigurators []func(*Options)
}

func (b *containerBuilder) ConfigureOptions(f func(*Options)) {
	b.optionsConfigurators = append(b.optionsConfigurators, f)
}

func (b *containerBuilder) Add(d ...*Descriptor) {
	b.descriptors = append(b.descriptors, d...)
}

func (b *containerBuilder) Remove(t reflect.Type) {
	descriptors := b.descriptors
	j := 0
	for _, d := range descriptors {
		if d.ServiceType != t {
			descriptors[j] = d
			j++
		}
	}
	b.descriptors = descriptors[:j]
}

func (b *containerBuilder) Contains(t reflect.Type) bool {
	for _, d := range b.descriptors {
		if d.ServiceType == t {
			return true
		}
	}
	return false
}

func (b *containerBuilder) builtInServices(c *container) {
	csf := c.CallSiteFactory

	csf.Add(ContainerType, &ContainerCallSite{})
	csf.Add(ScopeFactoryType, newConstantCallSite(ScopeFactoryType, c.Root))
	csf.Add(IsServiceType, newConstantCallSite(IsServiceType, csf))
}

func (b *containerBuilder) configureOptions(options *Options) {
	for _, f := range b.optionsConfigurators {
		f(options)
	}
}

func (b *containerBuilder) Build() Container {
	options := DefaultOptions()
	b.configureOptions(&options)

	c := &container{
		CallSiteFactory:           newCallSiteFactory(b.descriptors),
		realizedServices:          syncx.NewMap[reflect.Type, ServiceAccessor](),
		realizedLookupKeyServices: syncx.NewMap[string, ServiceAccessor](),
	}

	c.Root = newEngineScope(c, true)
	c.engine = c.createEngine()

	b.builtInServices(c)

	if options.ValidateScopes {
		c.callSiteValidator = newCallSiteValidator()
	}

	if options.ValidateOnBuild {
		errs := make([]error, 0)
		for _, d := range b.descriptors {
			if e := c.validateService(d); e != nil {
				errs = append(errs, e)
			}
		}

		if len(errs) > 0 {
			panic(&errorx.AggregateError{Errors: errs})
		}
	}

	return c
}

// Create a ContainerBuilder
func Builder() ContainerBuilder {
	return &containerBuilder{}
}

// New a descriptor with instance
func Instance[T any](instance any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	return NewInstanceDescriptor(reflectx.TypeOf[T](), instance)
}

// New a transient constructor descriptor
func Transient[T any](ctor any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	return NewConstructorDescriptor(reflectx.TypeOf[T](), Lifetime_Transient, ctor, implementedInterfaceTypes...)
}

// New a scoped constructor descriptor
func Scoped[T any](ctor any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	return NewConstructorDescriptor(reflectx.TypeOf[T](), Lifetime_Scoped, ctor, implementedInterfaceTypes...)
}

// New a singleton constructor descriptor
func Singleton[T any](ctor any, implementedInterfaceTypes ...reflect.Type) *Descriptor {
	return NewConstructorDescriptor(reflectx.TypeOf[T](), Lifetime_Singleton, ctor, implementedInterfaceTypes...)
}

// Add a transient service descriptor to the ContainerBuilder.
// T is the service type,
// cb is the ContainerBuilder,
// ctor is the constructor of the service T.
func AddTransient[T any](cb ContainerBuilder, ctor any, implementedInterfaceTypes ...reflect.Type) {
	cb.Add(Transient[T](ctor, implementedInterfaceTypes...))
}

// Add a scoped service descriptor to the ContainerBuilder.
// T is the service type,
// cb is the ContainerBuilder,
// ctor is the constructor of the service T.
// implementedInterfaceTypes is the implemented interface types of the service T.
func AddScoped[T any](cb ContainerBuilder, ctor any, implementedInterfaceTypes ...reflect.Type) {
	cb.Add(Scoped[T](ctor, implementedInterfaceTypes...))
}

// Add a scoped service descriptor to the ContainerBuilder.
// T is the service type,
// cb is the ContainerBuilder,
// ctor is the constructor of the service T.
// lookupKeys is the lookup keys of the service T.
// implementedInterfaceTypes is the implemented interface types of the service T.
func AddScopedWithLookupKeys[T any](cb ContainerBuilder,
	ctor any,
	lookupKeys []string,
	implementedInterfaceTypes ...reflect.Type) {
	descriptor := Scoped[T](ctor, implementedInterfaceTypes...)

	for _, key := range lookupKeys {
		hKey := hashTypeAndString(descriptor.ServiceType, key)
		descriptor.LookupKeys = append(descriptor.LookupKeys, hKey)
		for _, t := range implementedInterfaceTypes {
			hKey = hashTypeAndString(t, key)
			descriptor.LookupKeys = append(descriptor.LookupKeys, hKey)
		}
	}
	cb.Add(descriptor)
}

// Add a singleton service descriptor to the ContainerBuilder.
// T is the service type,
// cb is the ContainerBuilder,
// ctor is the constructor of the service T.
func AddSingleton[T any](cb ContainerBuilder, ctor any, implementedInterfaceTypes ...reflect.Type) {
	cb.Add(Singleton[T](ctor, implementedInterfaceTypes...))
}

// Add an instance service descriptor to the ContainerBuilder.
// T is the service type,
// cb is the ContainerBuilder,
// the instance must be assignable to the service T.
func AddInstance[T any](cb ContainerBuilder, instance any, implementedInterfaceTypes ...reflect.Type) {
	cb.Add(Instance[T](instance, implementedInterfaceTypes...))
}

// New a transient factory descriptor
func TransientFactory[T any](factory Factory) *Descriptor {
	return NewFactoryDescriptor(reflectx.TypeOf[T](), Lifetime_Transient, factory)
}

// New a scoped factory descriptor
func ScopedFactory[T any](factory Factory) *Descriptor {
	return NewFactoryDescriptor(reflectx.TypeOf[T](), Lifetime_Scoped, factory)
}

// New a singleton factory descriptor
func SingletonFactory[T any](factory Factory) *Descriptor {
	return NewFactoryDescriptor(reflectx.TypeOf[T](), Lifetime_Singleton, factory)
}

func AddTransientFactory[T any](cb ContainerBuilder, factory Factory) {
	cb.Add(TransientFactory[T](factory))
}

func AddScopedFactory[T any](cb ContainerBuilder, factory Factory) {
	cb.Add(ScopedFactory[T](factory))
}

func AddSingletonFactory[T any](cb ContainerBuilder, factory Factory) {
	cb.Add(SingletonFactory[T](factory))
}
