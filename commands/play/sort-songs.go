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
