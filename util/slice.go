package util

import "sort"

func ReverseSlice[T any](s []T) {
	sort.SliceStable(s, func(i, j int) bool {
		return i > j
	})
}

func ClipSlice[T any](s []T) []T {
	return s[:len(s):len(s)]
}
