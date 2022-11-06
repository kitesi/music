package play

import (
	"encoding/json"
	"log"
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

func getStoredTags(musicPath string) []Tag {
	var savedTags []Tag

	content, err := os.ReadFile(getTagPath(musicPath))

	if err == os.ErrNotExist {
		return savedTags
	} else if err != nil {
		log.Fatal("Error while opening tags file: ", err)
	} else {
		var payload []Tag
		err = json.Unmarshal(content, &payload)

		if err != nil {
			log.Fatal("Error during Unmarshal() of tags.json: ", err)
		}

		savedTags = payload
	}

	return savedTags
}

func changeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	var tag Tag

	tags := getStoredTags(musicPath)

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
