package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dozm/di/errorx"
	"github.com/dozm/di/syncx"
	"github.com/dozm/di/util"
)

type CallSiteKind byte

const (
	CallSiteKind_Constructor CallSiteKind = iota
	CallSiteKind_Constant
	CallSiteKind_Slice
	CallSiteKind_Container
	CallSiteKind_Scope
	CallSiteKind_Transient
	CallSiteKind_Singleton
)

type CallSite interface {
	ServiceType() reflect.Type
	Kind() CallSiteKind
	Value() any
	SetValue(any)
	Cache() ResultCache
}

//
type ConstantCallSite struct {
	serviceType reflect.Type
	value       any
}

func (cs *ConstantCallSite) Value() any {
	return cs.value
}

func (cs *ConstantCallSite) SetValue(v any) {
	cs.value = v
}

func (cs *ConstantCallSite) DefaultValue() any {
	return cs.value
}

func (cs *ConstantCallSite) ServiceType() reflect.Type {
	return cs.serviceType
}

func (cs *ConstantCallSite) Kind() CallSiteKind {
	return CallSiteKind_Constant
}

func (cs *ConstantCallSite) Cache() ResultCache {
	return NoneResultCache
}

func newConstantCallSite(serviceType reflect.Type, defaultValue any) *ConstantCallSite {
	return &ConstantCallSite{
		serviceType: serviceType,
		value:       defaultValue,
	}
}

//
type ConstructorCallSite struct {
	serviceType reflect.Type
	value       any
	Ctor        *ConstructorInfo
	Parameters  []CallSite
	cache       ResultCache
}

func (cs *ConstructorCallSite) Value() any {
	return cs.value
}

func (cs *ConstructorCallSite) SetValue(v any) {
	cs.value = v
}

func (cs *ConstructorCallSite) ServiceType() reflect.Type {
	return cs.serviceType
}

func (cs *ConstructorCallSite) Kind() CallSiteKind {
	return CallSiteKind_Constructor
}

func (cs *ConstructorCallSite) Cache() ResultCache {
	return cs.cache
}

func newConstructorCallSite(cache ResultCache, serviceType reflect.Type, ctor *ConstructorInfo, parameters []CallSite) *ConstructorCallSite {
	return &ConstructorCallSite{
		cache:       cache,
		serviceType: serviceType,
		Ctor:        ctor,
		Parameters:  parameters,
	}
}

//
type ContainerCallSite struct {
	value any
}

func (cs *ContainerCallSite) Value() any {
	return cs.value
}

func (cs *ContainerCallSite) SetValue(v any) {
	cs.value = v
}

func (cs *ContainerCallSite) ServiceType() reflect.Type {
	return ContainerType
}

func (cs *ContainerCallSite) Kind() CallSiteKind {
	return CallSiteKind_Container
}

func (cs *ContainerCallSite) Cache() ResultCache {
	return NoneResultCache
}

//
type SliceCallSite struct {
	serviceType reflect.Type
	Elem        reflect.Type
	CallSites   []CallSite
	cache       ResultCache
	value       any
}

func (cs *SliceCallSite) Value() any {
	return cs.value
}

func (cs *SliceCallSite) SetValue(v any) {
	cs.value = v
}

func (cs *SliceCallSite) Cache() ResultCache {
	return cs.cache
}

func (cs *SliceCallSite) ServiceType() reflect.Type {
	return cs.serviceType
}

func (cs *SliceCallSite) Kind() CallSiteKind {
	return CallSiteKind_Slice
}

func newSliceCallSite(cache ResultCache, elem reflect.Type, callSites []CallSite) *SliceCallSite {
	return &SliceCallSite{
		cache:       cache,
		Elem:        elem,
		CallSites:   callSites,
		serviceType: reflect.SliceOf(elem),
	}
}

//
type chainItem struct {
	Order int
	Ctor  *ConstructorInfo
}

type callSiteChain struct {
	items map[reflect.Type]chainItem
}

func (c *callSiteChain) CheckCircularDependency(serviceType reflect.Type) error {
	for k := range c.items {
		if k == serviceType {
			return c.createCircularDependencyError(serviceType)
		}
	}
	return nil
}

func (c *callSiteChain) Remove(serviceType reflect.Type) {
	delete(c.items, serviceType)
}

// the ctor can be nil when the serviceType is a slice
func (c *callSiteChain) Add(serviceType reflect.Type, ctor *ConstructorInfo) {
	c.items[serviceType] = chainItem{
		Order: len(c.items),
		Ctor:  ctor,
	}
}

func (c *callSiteChain) createCircularDependencyError(t reflect.Type) error {
	var sb strings.Builder
	sb.WriteString("a circular dependency was detected for the service of type '")
	sb.WriteString(t.String())
	sb.WriteString("'.")
	// TODO: add resolution path

	return &errorx.CircularDependencyError{Message: sb.String()}
}

func newCallSiteChain() *callSiteChain {
	return &callSiteChain{
		items: make(map[reflect.Type]chainItem),
	}
}

//

const DefaultSlot int = 0

type CallSiteFactory struct {
	descriptors      []*Descriptor
	callSiteCache    *syncx.Map[ServiceCacheKey, CallSite]
	descriptorLookup map[reflect.Type]descriptorCacheItem
	callSiteLockers  *syncx.LockMap
}

func (f *CallSiteFactory) Descriptors() []*Descriptor {
	return f.descriptors
}

func (f *CallSiteFactory) populate() {
	for _, descriptor := range f.descriptors {
		serviceType := descriptor.ServiceType
		cacheItem := f.descriptorLookup[serviceType]
		f.descriptorLookup[serviceType] = cacheItem.Add(descriptor)
	}
}

func (f *CallSiteFactory) GetCallSite(serviceType reflect.Type, chain *callSiteChain) (CallSite, error) {
	if site, ok := f.callSiteCache.Load(ServiceCacheKey{ServiceType: serviceType, Slot: DefaultSlot}); ok {
		return site, nil
	}

	return f.createCallSite(serviceType, chain)
}

func (f *CallSiteFactory) GetCallSiteByDescriptor(descriptor *Descriptor, chain *callSiteChain) (CallSite, error) {
	if descriptorCache, ok := f.descriptorLookup[descriptor.ServiceType]; ok {
		return f.tryCreateExact(
			descriptor,
			chain,
			descriptorCache.GetSlot(descriptor))
	}

	return nil, errors.New("descriptorLookup didn't contain requested descriptor")

}

func (f *CallSiteFactory) createCallSite(serviceType reflect.Type, chain *callSiteChain) (CallSite, error) {
	if err := chain.CheckCircularDependency(serviceType); err != nil {
		return nil, err
	}

	callSiteLocker := f.callSiteLockers.LoadOrCreate(serviceType)
	callSiteLocker.Lock()
	defer callSiteLocker.Unlock()

	if descriptor, ok := f.descriptorLookup[serviceType]; ok {
		return f.tryCreateExact(descriptor.Last(), chain, DefaultSlot)
	}

	if serviceType.Kind() == reflect.Slice {
		return f.createSlice(serviceType, chain)
	}

	return nil, &errorx.ServiceNotFound{ServiceType: serviceType}
}

func (f *CallSiteFactory) tryCreateExact(descriptor *Descriptor, chain *callSiteChain, slot int) (CallSite, error) {
	callSiteKey := ServiceCacheKey{descriptor.ServiceType, slot}
	callSite, ok := f.callSiteCache.Load(callSiteKey)
	if ok {
		return callSite, nil
	}

	cache := newResultCacheWithLifetime(descriptor.Lifetime, descriptor.ServiceType, slot)

	var err error
	if descriptor.Instance != nil {
		callSite = newConstantCallSite(descriptor.ServiceType, descriptor.Instance)
	} else if descriptor.Ctor != nil {
		callSite, err = f.createConstructorCallsite(cache, descriptor.ServiceType, descriptor.Ctor, chain)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, &errorx.InvalidDescriptor{ServiceType: descriptor.ServiceType}
	}

	f.callSiteCache.Store(callSiteKey, callSite)
	return callSite, nil
}

func (f *CallSiteFactory) createConstructorCallsite(cache ResultCache, serviceType reflect.Type, ctor *ConstructorInfo, chain *callSiteChain) (*ConstructorCallSite, error) {
	chain.Add(serviceType, ctor)
	defer chain.Remove(serviceType)

	if len(ctor.In) == 0 {
		return newConstructorCallSite(cache, serviceType, ctor, nil), nil
	}

	parameterCallSites, err := f.createArgumentCallSites(chain, ctor)
	if err != nil {
		return nil, err
	}

	return newConstructorCallSite(cache, serviceType, ctor, parameterCallSites), nil
}

func (f *CallSiteFactory) createArgumentCallSites(chain *callSiteChain, ctor *ConstructorInfo) ([]CallSite, error) {
	callSites := make([]CallSite, len(ctor.In))
	for i, t := range ctor.In {
		cs, err := f.GetCallSite(t, chain)
		if err != nil {
			return nil, err
		}
		callSites[i] = cs
	}
	return callSites, nil
}

func (f *CallSiteFactory) createSlice(serviceType reflect.Type, chain *callSiteChain) (CallSite, error) {
	if serviceType.Kind() != reflect.Slice {
		return nil, fmt.Errorf("service type '%v' is not slice", serviceType)
	}

	key := ServiceCacheKey{serviceType, DefaultSlot}
	if callSite, ok := f.callSiteCache.Load(key); ok {
		return callSite, nil
	}

	chain.Add(serviceType, nil)
	defer chain.Remove(serviceType)

	elementType := serviceType.Elem()
	cacheLocation := CacheLocation_Root
	callSites := make([]CallSite, 0)

	if descriptorCache, ok := f.descriptorLookup[elementType]; ok {
		num := descriptorCache.Num()
		for i := 0; i < num; i++ {
			cs, err := f.tryCreateExact(descriptorCache.Get(i), chain, num-i-1)
			if err != nil {
				return nil, err
			}

			cacheLocation = f.getCommonCacheLocation(cacheLocation, cs.Cache().Location)
			callSites = append(callSites, cs)
		}
	}

	resultCache := NoneResultCache
	if cacheLocation == CacheLocation_Scope || cacheLocation == CacheLocation_Root {
		resultCache = newResultCache(cacheLocation, key)
	}

	return newSliceCallSite(resultCache, elementType, util.ClipSlice(callSites)), nil
}

func (f *CallSiteFactory) Add(serviceType reflect.Type, callSite CallSite) {
	f.callSiteCache.Store(ServiceCacheKey{ServiceType: serviceType, Slot: DefaultSlot}, callSite)
}

// Determines if the specified service type is available from the ServiceProvider.
func (f *CallSiteFactory) IsService(serviceType reflect.Type) bool {
	if serviceType == nil {
		return false
	}

	if _, ok := f.descriptorLookup[serviceType]; ok {
		return true
	}

	if serviceType.Kind() == reflect.Slice {
		return true
	}

	return serviceType == ContainerType ||
		serviceType == ScopeFactoryType ||
		serviceType == IsServiceType
}

func (f *CallSiteFactory) getCommonCacheLocation(locationA CacheLocation, locationB CacheLocation) CacheLocation {
	if locationA > locationB {
		return locationA
	}
	return locationB

}

func newCallSiteFactory(descriptors []*Descriptor) *CallSiteFactory {
	d := make([]*Descriptor, len(descriptors))
	copy(d, descriptors)

	f := &CallSiteFactory{
		descriptors:      d,
		callSiteCache:    syncx.NewMap[ServiceCacheKey, CallSite](),
		descriptorLookup: make(map[reflect.Type]descriptorCacheItem),
		callSiteLockers:  &syncx.LockMap{},
	}

	f.populate()
	return f
}

type descriptorCacheItem struct {
	item  *Descriptor
	items []*Descriptor
}

func (dci descriptorCacheItem) Last() *Descriptor {
	if l := len(dci.items); l > 0 {
		return dci.items[l-1]
	}

	return dci.item
}

func (dci descriptorCacheItem) Num() int {
	if dci.item == nil {
		return 0
	}

	return 1 + len(dci.items)
}

func (dci descriptorCacheItem) Get(index int) *Descriptor {
	if index >= dci.Num() {
		panic("index out of range")
	}

	if index == 0 {
		return dci.item
	}

	return dci.items[index-1]
}

func (dci descriptorCacheItem) GetSlot(descriptor *Descriptor) int {
	if descriptor == dci.item {
		return dci.Num() - 1
	}

	if l := len(dci.items); l > 0 {
		for i := range dci.items {
			if descriptor == dci.items[i] {
				return l - (i + 1)
			}
		}
	}

	panic(errors.New("descriptor not exist"))
}

func (dci descriptorCacheItem) Add(descriptor *Descriptor) descriptorCacheItem {
	var newCacheItem descriptorCacheItem
	if dci.item == nil {
		newCacheItem.item = descriptor
	} else {
		newCacheItem.item = dci.item
		newCacheItem.items = append(dci.items, descriptor)
	}
	return newCacheItem
}
