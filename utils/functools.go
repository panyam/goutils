package utils

import (
	"sync"
)

func Map[T any, U any](items []T, mapfunc func(T) U) (out []U) {
	for _, item := range items {
		out = append(out, mapfunc(item))
	}
	return
}

func Filter[T any](items []T, filtfunc func(T) bool) (out []T) {
	for _, item := range items {
		if filtfunc(item) {
			out = append(out, item)
		}
	}
	return
}

func BatchGet[T any](ids []string, reqMaker func(string) (T, error)) (out map[string]T) {
	var wg sync.WaitGroup
	var respMutex sync.Mutex
	out = make(map[string]T)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			resp, err := reqMaker(id)
			if err == nil {
				respMutex.Lock()
				out[id] = resp
				respMutex.Unlock()
			}
		}(id)
	}
	wg.Wait()
	return
}
