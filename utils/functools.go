package utils

func Map[T any, U any](items []T, mapfunc func(T) U) (out []U) {
	for _, item := range items {
		out = append(out, mapfunc(item))
	}
	return
}
