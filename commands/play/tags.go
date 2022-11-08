package play

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Tag struct {
	Name  string   `json:"name"`
	Songs []string `json:"songs"`
}

func getTagPath(musicPath string) string {
	return filepath.Join(musicPath, "tags.json")
}

func getStoredTags(musicPath string) ([]Tag, error) {
	var savedTags []Tag

	content, err := os.ReadFile(getTagPath(musicPath))

	if err == nil {
		var payload []Tag
		err = json.Unmarshal(content, &payload)

		if err != nil {
			return nil, err
		}

		savedTags = payload
	} else if errors.Is(err, fs.ErrNotExist) {
		return savedTags, nil
	} else {
		return nil, err
	}

	return savedTags, nil
}

func changeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	var tag Tag

	tags, err := getStoredTags(musicPath)

	if err != nil {
		return err
	}

	for _, t := range tags {
		if t.Name == tagName {
			tag = t
		}
	}

	if tag.Name == "" {
		tag = Tag{Name: tagName, Songs: []string{}}
	} else if !shouldAppend {
		tag.Songs = []string{}
	}

	for _, song := range songs {
		bareName := strings.Replace(song, musicPath, "", 1)

		if includes(tag.Songs, bareName) {
			tag.Songs = append(tag.Songs, bareName)
		}
	}

	tagsString, err := json.Marshal(tags)

	if err != nil {
		return err
	}

	err = os.WriteFile(getTagPath(musicPath), tagsString, 0666)
	return err
}
