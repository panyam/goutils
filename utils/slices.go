package utils

/**
 * A method to insert an item at an index in a slice.
 */
func Insert[T any](list []T, item T, index int) []T {
	return append(list[:index], append([]T{item}, list[index:]...)...)
}
