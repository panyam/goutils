package utils

import (
	"sync"
)

func IndexOf[T any](items []T, matchfunc func(T) bool) int {
	for index, item := range items {
		if matchfunc(item) {
			return index
		}
	}
	return -1
}

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

func MapKeys[K comparable, V any](input map[K]V) []K {
	out := make([]K, 0, len(input))
	for k := range input {
		out = append(out, k)
	}
	return out
}

func MapValues[V any, K comparable](input map[K]V) []V {
	out := make([]V, 0, len(input))
	for _, v := range input {
		out = append(out, v)
	}
	return out
}
