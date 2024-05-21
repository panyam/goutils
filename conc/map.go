package conc

import "sync"

// A synchronized map with read and update lock capabilities
type Map[K comparable, V any] struct {
	rwmutex sync.RWMutex
	store   map[K]V
}

// Creates a new lockable map
func NewMap[K comparable, V any]() (out *Map[K, V]) {
	out = &Map[K, V]{
		store: make(map[K]V),
	}
	return
}

// Obtains a read lock on the map
func (m *Map[K, V]) RLock() {
	m.rwmutex.RLock()
}

// Relinquishes a read lock on the map
func (m *Map[K, V]) RUnlock() {
	m.rwmutex.RUnlock()
}

// Obtains a write lock on the map
func (m *Map[K, V]) Lock() {
	m.rwmutex.Lock()
}

// Relinquishes a write lock on the map
func (m *Map[K, V]) Unlock() {
	m.rwmutex.Unlock()
}

// Check if the map contains an entry by key.
// Optionally obtains a read lock if requested.
// A lock can be skipped if a lock was already obtained over a larger transaction.
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

// Gets the value by a given key.
// Optionally obtains a read lock if requested.
// A lock can be skipped if a lock was already obtained over a larger transaction.
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

// Sets the value by a given key.
// Optionally obtains a write lock if requested.
// The lock can be skipped if a lock was already obtained over a larger transaction.
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

// Deletes the value by a given key.
// Optionally obtains a write lock if requested.
// The lock can be skipped if a lock was already obtained over a larger transaction.
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

// Iterates over the items in this map.
// Optionally obtains a read lock if requested.
// The lock can be skipped if a lock was already obtained over a larger transaction.
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

// Obtains a read lock over the map to perform a list of read actions
func (m *Map[K, V]) View(actions func()) {
	m.rwmutex.RLock()
	if m.store == nil {
		m.store = make(map[K]V)
	}
	actions()
	m.rwmutex.RUnlock()
}

// Obtains a write lock over the map to perform a list of udpate actions
func (m *Map[K, V]) Update(actions func(items map[K]V)) {
	m.rwmutex.Lock()
	if m.store == nil {
		m.store = make(map[K]V)
	}
	actions(m.store)
	m.rwmutex.Unlock()
}

// Locks the map to check for key membership
func (m *Map[K, V]) Has(k K) bool {
	return m.LHas(k, true)
}

// Locks the map to get the value of a given key.
func (m *Map[K, V]) Get(k K) (V, bool) {
	return m.LGet(k, true)
}

// Locks the map to set the value of a given key.
func (m *Map[K, V]) Set(k K, v V) {
	m.LSet(k, v, true)
}

// Locks the map to delete a given key.
func (m *Map[K, V]) Delete(k K) {
	m.LDelete(k, true)
}

// Locks the map to range of keys/values in this map
func (m *Map[K, V]) Range(meth func(K, V) bool) {
	m.LRange(true, meth)
}
