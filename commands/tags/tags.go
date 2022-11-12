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
	"time"

	arrayUtils "github.com/kitesi/music/array-utils"
	"github.com/kitesi/music/editor"
	stringUtils "github.com/kitesi/music/string-utils"
	"github.com/spf13/cobra"
)

type TagsCommandArgs struct {
	edit         bool
	shouldDelete bool
	musicPath    string
}

type Tag struct {
	Songs        []string `json:"songs"`
	CreationTime int64    `json:"creation_time"`
	ModifiedTime int64    `json:"modified_time"`
}

type Tags map[string]Tag

func GetTagPath(musicPath string) string {
	return filepath.Join(musicPath, "tags.json")
}

func GetStoredTags(musicPath string) (Tags, error) {
	storedTags := Tags{}

	content, err := os.ReadFile(GetTagPath(musicPath))

	if err == nil {
		err = json.Unmarshal(content, &storedTags)

		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		return storedTags, nil
	} else {
		return nil, err
	}

	return storedTags, nil
}

func Setup(rootCmd *cobra.Command) {
	args := TagsCommandArgs{}

	tagsCmd := &cobra.Command{
		Use:   "tags [tags..]",
		Short: "Manage tags",
		Long:  "Manage tags. Lists all the tags by default. If a tag is provided, this will list all the songs in that tag.",
		Run: func(cmd *cobra.Command, positional []string) {
			if err := tagsCommandRunner(&args, positional); err != nil {
				log.SetFlags(0)
				log.Fatal(err)
			}
		},
	}

	tagsCmd.Flags().BoolVarP(&args.edit, "editor", "e", false, "edit tags.json or a specific tag with $EDITOR")
	tagsCmd.Flags().BoolVarP(&args.shouldDelete, "delete", "d", false, "delete a tag")
	tagsCmd.Flags().StringVarP(&args.musicPath, "music-path", "m", "", "the music path to use")

	rootCmd.AddCommand(tagsCmd)
}

func tagsCommandRunner(args *TagsCommandArgs, positional []string) error {
	if args.shouldDelete {
		if args.edit {
			return errors.New("can't have --delete and --editor together")
		}

		if len(positional) == 0 {
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

	if len(positional) == 0 {
		if args.edit {
			_, err := editor.EditFile(GetTagPath(args.musicPath))
			return err
		}

		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return err
		}

		for k := range storedTags {
			fmt.Println(k)
		}

		return nil
	}

	storedTags, err := GetStoredTags(args.musicPath)

	if err != nil {
		return err
	}

	for _, requestedTagName := range positional {
		tag, ok := storedTags[requestedTagName]

		if args.edit {
			content, err := editor.CreateAndModifyTemp("", requestedTagName+"-*.txt", strings.Join(tag.Songs, "\n"))

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			if err = ChangeSongsInTag(args.musicPath, requestedTagName, strings.Split(content, "\n"), false); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		if !ok {
			fmt.Fprintf(os.Stderr, "Error: tag \"%s\" does not exist\n", requestedTagName)
			continue
		}

		if args.shouldDelete {
			delete(storedTags, requestedTagName)

			if err := updateTagsFile(&storedTags, args.musicPath); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		fmt.Printf("Name: %s, Amount: %d, Creation: %s, Modified: %s\n", requestedTagName, len(tag.Songs), formatTime(tag.ModifiedTime), formatTime(tag.CreationTime))
		fmt.Println(strings.Join(tag.Songs, "\n"))

	}
	return nil
}

func formatTime(timeStamp int64) string {
	return time.Unix(timeStamp, 0).Format("2006-01-02 15:04:05")
}

func ChangeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	storedTags, err := GetStoredTags(musicPath)

	if err != nil {
		return err
	}

	tag, ok := storedTags[tagName]

	now := time.Now().Unix()

	if !ok {
		tag = Tag{CreationTime: now, ModifiedTime: now, Songs: arrayUtils.FilterEmptystrings(songs)}
	} else if !shouldAppend {
		tag.Songs = arrayUtils.FilterEmptystrings(songs)
	} else {
		for _, song := range songs {
			if song != "" && !arrayUtils.Includes(tag.Songs, song) {
				tag.Songs = append(tag.Songs, song)
			}
		}
	}

	tag.ModifiedTime = now
	storedTags[tagName] = tag

	return updateTagsFile(&storedTags, musicPath)
}

func updateTagsFile(tags *Tags, musicPath string) error {
	tagsString, err := json.Marshal(tags)

	if err != nil {
		return err
	}

	return os.WriteFile(GetTagPath(musicPath), tagsString, 0666)
}
