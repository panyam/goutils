package http

import (
	"net/http"
	"sync"
)

type RequestVarMap struct {
	keyLock sync.RWMutex
	keys    map[*http.Request]map[string]any
}

func (r *RequestVarMap) SetKey(req *http.Request, key string, value any) {
	r.keyLock.Lock()
	defer r.keyLock.Unlock()
	if r.keys == nil {
		r.keys = make(map[*http.Request]map[string]any)
	}
	if r.keys[req] == nil {
		r.keys[req] = make(map[string]any)
	}
	r.keys[req][key] = value
}

func (r *RequestVarMap) GetKey(req *http.Request, key string) (value any) {
	r.keyLock.RLock()
	defer r.keyLock.RUnlock()
	if r.keys == nil {
		r.keys = make(map[*http.Request]map[string]any)
	}
	if r.keys[req] == nil {
		r.keys[req] = make(map[string]any)
	}
	return r.keys[req][key]
}
