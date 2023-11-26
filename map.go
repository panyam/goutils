package conc

import "sync"

type Map[K comparable, V any] struct {
	rwmutex sync.RWMutex
	store   map[K]V
}

func NewMap[K comparable, V any](out *Map[K, V]) {
	out = &Map[K, V]{
		store: make(map[K]V),
	}
	return
}

func (m *Map[K, V]) Get(k K, lock bool) V {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	return m.store[k]
}

func (m *Map[K, V]) Set(k K, v V, lock bool) {
	if lock {
		m.rwmutex.Lock()
		defer m.rwmutex.Unlock()
	}
	m.store[k] = v
}

func (m *Map[K, V]) Delete(k K, lock bool) {
	if lock {
		m.rwmutex.Lock()
		defer m.rwmutex.Unlock()
	}
	delete(m.store, k)
}

func (m *Map[K, V]) Range(lock bool, meth func(K, V) bool) {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	for k, v := range m.store {
		if !meth(k, v) {
			break
		}
	}
}

func (m *Map[K, V]) View(actions func()) {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	actions()
}

func (m *Map[K, V]) Update(actions func()) {
	m.rwmutex.Lock()
	defer m.rwmutex.Unlock()
	actions()
}
