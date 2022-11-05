package play

import (
	"sort"
)

func sortByNew(songs []Song, requestedTimeStat string) {
	sort.Slice(songs, func(i, j int) bool {
		if requestedTimeStat == "a" {
			return songs[i].stat.AccessTime().After(songs[j].stat.AccessTime())
		} else if requestedTimeStat == "c" {
			return songs[i].stat.ChangeTime().After(songs[j].stat.ChangeTime())
		}

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

func includes[T comparable](arr []T, item T) bool {
	return some(arr, func(i T) bool {
		return i == item
	})
}
