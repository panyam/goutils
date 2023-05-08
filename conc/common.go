package conc

func IDFunc[T any](input T) T {
	return input
}

type Message[T any] struct {
	Value  T
	Error  error
	Source interface{}
}
