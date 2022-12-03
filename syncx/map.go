package syncx

import (
	"sync"
)

type Map[TK any, TV any] struct {
	data sync.Map
}

func (m *Map[TK, TV]) Store(key TK, value TV) {
	m.data.Store(key, value)
}

func (m *Map[TK, TV]) Delete(key TK) {
	m.data.Delete(key)
}

func (m *Map[TK, TV]) Load(key TK) (TV, bool) {
	v, ok := m.data.Load(key)
	var v2 TV
	if ok {
		v2, ok = v.(TV)

	}
	return v2, ok
}

func (m *Map[TK, TV]) LoadOrStore(key TK, value TV) (TV, bool) {
	v, ok := m.data.LoadOrStore(key, value)
	return v.(TV), ok
}

func (m *Map[TK, TV]) LoadOrCreate(key TK, valueFactory func(TK) TV) (TV, bool) {
	if v, loaded := m.data.Load(key); loaded {
		return v.(TV), true
	}

	v, loaded := m.data.LoadOrStore(key, valueFactory(key))
	return v.(TV), loaded
}

func NewMap[TK any, TV any]() *Map[TK, TV] {
	return &Map[TK, TV]{}
}
