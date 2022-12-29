package utils

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
