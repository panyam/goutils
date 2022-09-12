package conc

func IDFunc[T any](input T) T {
	return input
}

type ValueOrError[T any] struct {
	Value T
	Error error
}
