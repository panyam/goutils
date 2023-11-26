package conc

import "sync"

type Map[K comparable, V any] struct {
	rwmutex sync.RWMutex
	store   map[K]V
}

func NewMap[K comparable, V any]() (out *Map[K, V]) {
	out = &Map[K, V]{
		store: make(map[K]V),
	}
	return
}

func (m *Map[K, V]) RLock() {
	m.rwmutex.RLock()
}

func (m *Map[K, V]) RUnlock() {
	m.rwmutex.RUnlock()
}

func (m *Map[K, V]) Lock() {
	m.rwmutex.Lock()
}

func (m *Map[K, V]) Unlock() {
	m.rwmutex.Unlock()
}

func (m *Map[K, V]) Has(k K, lock bool) bool {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	_, exists := m.store[k]
	return exists
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
	actions()
	m.rwmutex.RUnlock()
}

func (m *Map[K, V]) Update(actions func()) {
	m.rwmutex.Lock()
	actions()
	m.rwmutex.Unlock()
}
