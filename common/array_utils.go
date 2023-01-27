package common

func MapArray[T, V any](input []T, mapper func(T) V) []V {
	result := make([]V, len(input))
	for i, v := range input {
		result[i] = mapper(v)
	}
	return result
}
