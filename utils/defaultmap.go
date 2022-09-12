package utils

type DefaultMap[V interface{}] struct {
	entries map[string]V
	creator func(k string) V
}

func NewDefaultMap[V any](creator func(key string) V) *DefaultMap[V] {
	return &DefaultMap[V]{
		entries: make(map[string]V),
		creator: creator,
	}
}

func (tm *DefaultMap[V]) Get(key string) (V, bool) {
	value, ok := tm.entries[key]
	return value, ok
}

func (tm *DefaultMap[V]) Ensure(key string) V {
	if _, ok := tm.entries[key]; !ok {
		tm.entries[key] = tm.creator(key)
	}
	return tm.entries[key]
}
