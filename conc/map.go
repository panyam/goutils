package conc

import "sync"

type Map[K comparable, V any] struct {
	rwmutex sync.RWMutex
	items   map[K]V
}

func NewMap[K comparable, V any]() (out *Map[K, V]) {
	out = &Map[K, V]{
		items: make(map[K]V),
	}
	return
}

func (m *Map[K, V]) Has(k K) bool {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	_, exists := m.items[k]
	return exists
}

func (m *Map[K, V]) Get(k K) (V, bool) {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	out, ok := m.items[k]
	return out, ok
}

func (m *Map[K, V]) Set(k K, v V) {
	m.rwmutex.Lock()
	defer m.rwmutex.Unlock()
	m.items[k] = v
}

func (m *Map[K, V]) Delete(k K) {
	m.rwmutex.Lock()
	defer m.rwmutex.Unlock()
	delete(m.items, k)
}

func (m *Map[K, V]) Range(meth func(K, V) bool) {
	m.rwmutex.RLock()
	defer m.rwmutex.RUnlock()
	for k, v := range m.items {
		if !meth(k, v) {
			break
		}
	}
}

func (m *Map[K, V]) View(actions func(items map[K]V)) {
	m.rwmutex.RLock()
	actions(m.items)
	m.rwmutex.RUnlock()
}

func (m *Map[K, V]) Update(actions func(items map[K]V)) {
	m.rwmutex.Lock()
	actions(m.items)
	m.rwmutex.Unlock()
}
