package di

import "reflect"

// callsite result cache location
type CacheLocation byte

const (
	CacheLocation_Root CacheLocation = iota
	CacheLocation_Scope
	CacheLocation_Dispose
	CacheLocation_None
)

var NoneResultCache = newResultCache(CacheLocation_None, EmptyServiceCacheKey)

type ServiceCacheKey struct {
	// Type of service being cached
	ServiceType reflect.Type

	// Reverse index of the service when resolved in slice where default instance gets slot 0.
	Slot int
}

var EmptyServiceCacheKey = ServiceCacheKey{nil, 0}

// callsite result cache
type ResultCache struct {
	Location CacheLocation
	Key      ServiceCacheKey
}

func newResultCache(loc CacheLocation, key ServiceCacheKey) ResultCache {
	return ResultCache{
		Location: loc,
		Key:      key,
	}
}

func newResultCacheWithLifetime(lifetime Lifetime, typ reflect.Type, slot int) ResultCache {
	loc := CacheLocation_None
	switch lifetime {
	case Lifetime_Singleton:
		loc = CacheLocation_Root
	case Lifetime_Scoped:
		loc = CacheLocation_Scope
	case Lifetime_Transient:
		loc = CacheLocation_Dispose
	}

	return ResultCache{
		Location: loc,
		Key:      ServiceCacheKey{typ, slot},
	}
}
