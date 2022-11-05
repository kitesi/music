package play

import "sort"

func sortByNew(songs []Song) {
	sort.Slice(songs, func(i, j int) bool {
		return songs[i].stat.ModTime().After(songs[j].stat.ModTime())
	})
}

func every[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if !validator(el) {
			return false
		}
	}

	return true
}

func some[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if validator(el) {
			return true
		}
	}

	return false
}
