package syncx

import (
	"sync"
)

type LockMap struct {
	locks sync.Map
}

func (lm *LockMap) LoadOrCreate(key any) *sync.Mutex {
	v, ok := lm.locks.Load(key)
	if !ok {
		v, _ = lm.locks.LoadOrStore(key, &sync.Mutex{})
	}
	m, _ := v.(*sync.Mutex)
	return m
}

func (lm *LockMap) Lock(key any) {
	v, ok := lm.locks.Load(key)
	if !ok {
		v, _ = lm.locks.LoadOrStore(key, &sync.Mutex{})
	}
	m, _ := v.(*sync.Mutex)
	m.Lock()
}

func (lm *LockMap) Unlock(key any) {
	v, ok := lm.locks.Load(key)
	if !ok {
		panic("locker not exist")
	}
	m, _ := v.(*sync.Mutex)
	m.Unlock()
}

func (lm *LockMap) Len() int {
	l := 0
	lm.locks.Range(func(_, _ any) bool {
		l++
		return true
	})
	return l
}
