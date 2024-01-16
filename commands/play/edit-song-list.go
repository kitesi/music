package play

import (
	"strings"

	"github.com/kitesi/music/editor"
)

func editSongList(songs []string) ([]string, error) {
	content, err := editor.CreateAndModifyTemp("", "music-playlist-*.txt", strings.Join(songs, "\n"))

	if err != nil {
		return nil, err
	}

	editedSongs := strings.Split(content, "\n")
	filteredEditedSongs := make([]string, 0, len(editedSongs))

	for _, s := range editedSongs {
		if s != "" {
			filteredEditedSongs = append(filteredEditedSongs, s)
		}
	}

	return filteredEditedSongs, nil
}
