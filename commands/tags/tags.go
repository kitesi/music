package tags

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	arrayUtils "github.com/kitesi/music/array-utils"
	"github.com/kitesi/music/editor"
	stringUtils "github.com/kitesi/music/string-utils"
	"github.com/spf13/cobra"
)

type TagsCommandArgs struct {
	editor       bool
	shouldDelete bool
	musicPath    string
}

type Tag struct {
	Name  string   `json:"name"`
	Songs []string `json:"songs"`
}

func GetTagPath(musicPath string) string {
	return filepath.Join(musicPath, "tags.json")
}

func GetStoredTags(musicPath string) ([]Tag, error) {
	var savedTags []Tag

	content, err := os.ReadFile(GetTagPath(musicPath))

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

func Setup(rootCmd *cobra.Command) {
	args := TagsCommandArgs{}

	tagsCmd := &cobra.Command{
		Use:   "tags [tag]",
		Short: "Manage tags",
		Long:  "Lists all the tags be default. If a tag is provided, this will list all the songs in that list.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, positional []string) {
			if err := tagsCommandRunner(&args, positional); err != nil {
				log.SetFlags(0)
				log.Fatal(err)
			}
		},
	}

	tagsCmd.Flags().BoolVarP(&args.editor, "editor", "e", false, "edit tags or tag with $EDITOR")
	tagsCmd.Flags().BoolVarP(&args.shouldDelete, "delete", "d", false, "delete tag")
	tagsCmd.Flags().StringVarP(&args.musicPath, "music-path", "m", "", "music path")

	rootCmd.AddCommand(tagsCmd)
}

func tagsCommandRunner(args *TagsCommandArgs, positional []string) error {
	requestedTagName := ""

	if len(positional) > 0 {
		requestedTagName = positional[0]
	}

	if args.shouldDelete {
		if args.editor {
			return errors.New("can't have --delete and --editor together")
		}

		if requestedTagName == "" {
			return errors.New("can't use --delete without a tag")
		}
	}

	if args.musicPath == "" {
		defaultMusicPath, err := stringUtils.GetDefaultMusicPath()

		if err != nil {
			return err
		}

		args.musicPath = defaultMusicPath
	}

	if requestedTagName == "" {
		if args.editor {
			_, err := editor.EditFile(GetTagPath(args.musicPath))
			return err
		}

		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return err
		}

		for _, t := range storedTags {
			fmt.Println(t.Name)
		}

		return nil
	}

	storedTags, err := GetStoredTags(args.musicPath)

	if err != nil {
		return err
	}

	var tag *Tag
	tagIndex := -1

	for i, t := range storedTags {
		if t.Name == requestedTagName {
			tag = &t
			tagIndex = i
			break
		}
	}

	if args.shouldDelete {
		storedTags = append(storedTags[:tagIndex], storedTags[tagIndex+1:]...)
		return updateTagsFile(&storedTags, args.musicPath)
	}

	if args.editor {
		if tag == nil {
			tag = &Tag{}
		}

		content, err := editor.CreateAndModifyTemp("", requestedTagName+"-*.txt", strings.Join(tag.Songs, "\n"))

		if err != nil {
			return err
		}

		tag.Songs = arrayUtils.FilterEmptystrings(strings.Split(content, "\n"))

		if tag.Name == "" {
			tag.Name = requestedTagName
			storedTags = append(storedTags, *tag)
		}

		return updateTagsFile(&storedTags, args.musicPath)
	}

	if tag == nil {
		return fmt.Errorf("Tag \"%s\" does not exist", requestedTagName)
	}

	fmt.Println(strings.Join(tag.Songs, "\n"))

	return nil
}

func ChangeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	var tag *Tag
	storedTags, err := GetStoredTags(musicPath)
	tagIndex := -1

	if err != nil {
		return err
	}

	for i, t := range storedTags {
		if t.Name == tagName {
			tag = &t
			tagIndex = i
			break
		}
	}

	if tag == nil {
		storedTags = append(storedTags, Tag{Name: tagName, Songs: arrayUtils.FilterEmptystrings(songs)})
	} else {
		if !shouldAppend {
			tag.Songs = make([]string, 0, len(songs))
		}

		for _, song := range songs {
			if song != "" && !arrayUtils.Includes(tag.Songs, song) {
				tag.Songs = append(tag.Songs, song)
			}
		}

		storedTags[tagIndex] = *tag
	}

	return updateTagsFile(&storedTags, musicPath)
}

func updateTagsFile(tags *[]Tag, musicPath string) error {
	tagsString, err := json.Marshal(tags)

	if err != nil {
		return err
	}

	return os.WriteFile(GetTagPath(musicPath), tagsString, 0666)
}
