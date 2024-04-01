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

func (m *Map[K, V]) LHas(k K, lock bool) bool {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	if m.store == nil {
		m.store = make(map[K]V)
	}
	_, exists := m.store[k]
	return exists
}

func (m *Map[K, V]) LGet(k K, lock bool) (V, bool) {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	if m.store == nil {
		m.store = make(map[K]V)
	}
	out, ok := m.store[k]
	return out, ok
}

func (m *Map[K, V]) LSet(k K, v V, lock bool) {
	if lock {
		m.rwmutex.Lock()
		defer m.rwmutex.Unlock()
	}
	if m.store == nil {
		m.store = make(map[K]V)
	}
	m.store[k] = v
}

func (m *Map[K, V]) LDelete(k K, lock bool) {
	if lock {
		m.rwmutex.Lock()
		defer m.rwmutex.Unlock()
	}
	if m.store == nil {
		m.store = make(map[K]V)
	}
	delete(m.store, k)
}

func (m *Map[K, V]) LRange(lock bool, meth func(K, V) bool) {
	if lock {
		m.rwmutex.RLock()
		defer m.rwmutex.RUnlock()
	}
	if m.store == nil {
		m.store = make(map[K]V)
	}
	for k, v := range m.store {
		if !meth(k, v) {
			break
		}
	}
}

func (m *Map[K, V]) View(actions func()) {
	m.rwmutex.RLock()
	if m.store == nil {
		m.store = make(map[K]V)
	}
	actions()
	m.rwmutex.RUnlock()
}

func (m *Map[K, V]) Update(actions func(items map[K]V)) {
	m.rwmutex.Lock()
	if m.store == nil {
		m.store = make(map[K]V)
	}
	actions(m.store)
	m.rwmutex.Unlock()
}

func (m *Map[K, V]) Has(k K) bool {
	return m.LHas(k, true)
}

func (m *Map[K, V]) Get(k K) (V, bool) {
	return m.LGet(k, true)
}

func (m *Map[K, V]) Set(k K, v V) {
	m.LSet(k, v, true)
}

func (m *Map[K, V]) Delete(k K) {
	m.LDelete(k, true)
}

func (m *Map[K, V]) Range(meth func(K, V) bool) {
	m.LRange(true, meth)
}
